// Copyright 2022 Listware

package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"git.fg-tech.ru/listware/cmdb/pkg/cmdb/qdsl"
	"git.fg-tech.ru/listware/proto/sdk/pbflink"
	"git.fg-tech.ru/listware/proto/sdk/pbtypes"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	forwardedHeader = "X-Forwarded-Host"
)

var (
	noopFromFunction = &pbflink.FromFunction{
		Response: &pbflink.FromFunction_InvocationResult{
			InvocationResult: &pbflink.FromFunction_InvocationResponse{},
		},
	}
)

type Proxy struct {
	addr         string
	reverseProxy *httputil.ReverseProxy

	log *zap.SugaredLogger

	ctx    context.Context
	cancel context.CancelFunc

	noopData []byte
}

func New(opts ...Opt) (p *Proxy, err error) {
	l, err := zap.NewProduction()
	if err != nil {
		return
	}
	p = &Proxy{
		log: l.Sugar(),
	}
	p.reverseProxy = &httputil.ReverseProxy{
		Director:       p.directorFn,
		ModifyResponse: p.modifyResponse,
		ErrorHandler:   p.errorHandler,
	}

	if p.noopData, err = proto.Marshal(noopFromFunction); err != nil {
		return
	}

	return p, p.Configure(opts...)
}

func (p *Proxy) Configure(opts ...Opt) (err error) {
	for _, opt := range opts {
		if err = opt(p); err != nil {
			return
		}
	}
	return
}

func (p *Proxy) Run(ctx context.Context) (err error) {
	r := mux.NewRouter()
	r.HandleFunc("/{type}", handler(p.reverseProxy))

	srv := &http.Server{
		Addr:    p.addr,
		Handler: r,
	}

	p.ctx, p.cancel = context.WithCancel(ctx)

	p.log.Debugf("listen & serve on: %s", p.addr)

	go func() {
		if err = srv.ListenAndServe(); err != nil {
			p.log.Errorf("failed: %s", err.Error())
			p.cancel()
		}
	}()
	<-p.ctx.Done()
	if err != nil {
		return
	}
	return srv.Shutdown(p.ctx)
}

func (p *Proxy) directorFn(r *http.Request) {
	if r.Method != http.MethodPost {
		p.log.Errorf("only '%s' method allowed: %s", http.MethodPost, r.Method)
		return
	}

	functionType := mux.Vars(r)["type"]

	body, err := io.ReadAll(r.Body)
	if err != nil {
		p.log.Error(err.Error())
		return
	}

	var toFunction pbflink.ToFunction
	if err = proto.Unmarshal(body, &toFunction); err != nil {
		p.log.Error(err.Error())
		return
	}

	batch := toFunction.GetInvocation()
	tgt := batch.GetTarget()

	if r.URL, err = p.queryURL(functionType, tgt.GetId()); err != nil {
		p.log.Error(err.Error())
		return
	}

	if body, err = proto.Marshal(&toFunction); err != nil {
		p.log.Error(err.Error())
		return
	}

	r.Body = io.NopCloser(bytes.NewBuffer(body))
	r.ContentLength = int64(len(body))

	r.Header.Add(forwardedHeader, r.Host)
	r.Host = r.URL.Host
}

func (p *Proxy) modifyResponse(r *http.Response) (err error) {
	if r.StatusCode != http.StatusOK {
		return p.noopResponse(r)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		p.log.Error(err.Error())
		return p.noopResponse(r)
	}

	var fromFunction pbflink.FromFunction
	if err = proto.Unmarshal(body, &fromFunction); err != nil {
		return p.noopResponse(r)
	}

	r.Body = io.NopCloser(bytes.NewBuffer(body))
	r.ContentLength = int64(len(body))
	r.Header.Set("Content-Length", fmt.Sprint(r.ContentLength))
	return
}

func (p *Proxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusOK)
	w.Write(p.noopData)
}

func (p *Proxy) noopResponse(r *http.Response) (err error) {
	r.Body = io.NopCloser(bytes.NewBuffer(p.noopData))
	r.ContentLength = int64(len(p.noopData))
	r.StatusCode = http.StatusOK

	r.Header.Del("X-Content-Type-Options")

	r.Header.Set("Content-Length", fmt.Sprint(r.ContentLength))
	r.Header.Set("Content-Type", "application/octet-stream")
	return
}

// FIXME add r.URL to DNS instead cmdb qdsl
// r.Host = functionType
// functionType = from dns
func (p *Proxy) queryURL(functionType, id string) (u *url.URL, err error) {
	query := fmt.Sprintf("*[?@._id == '%s'?].%s", id, functionType)

	elems, err := qdsl.Qdsl(p.ctx, query, qdsl.WithLink())
	if err != nil {
		return
	}
	elemslen := len(elems)
	if elemslen == 0 {
		err = fmt.Errorf("unknown function implementation: %s", query)
		return
	}
	if elemslen > 1 {
		err = fmt.Errorf("multiple implementation of function: %s", query)
		return
	}
	var link pbtypes.FunctionRoute
	if err = json.Unmarshal(elems[0].Link, &link); err != nil {
		return
	}
	return url.Parse(link.Url)
}

func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		p.ServeHTTP(w, r)
	}
}
