// Copyright 2023 NJWS, INC
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

	"git.fg-tech.ru/listware/cmdb/pkg/cmdb/finder"
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
	port         int
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
	r.HandleFunc("/proxy/{type}", handler(p.reverseProxy))
	r.HandleFunc("/readyz", p.readyz)
	r.HandleFunc("/livez", p.livez)

	addr := fmt.Sprintf(":%d", p.port)

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	p.ctx, p.cancel = context.WithCancel(ctx)

	p.log.Debugf("listen & serve on: %s", srv.Addr)

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

func (p *Proxy) readyz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (p *Proxy) livez(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (p *Proxy) directorFn(r *http.Request) {
	p.log.Debugf("director: (%s)", r.Method)

	if r.Method != http.MethodPost {
		p.log.Errorf("only '%s' method allowed: %s", http.MethodPost, r.Method)
		return
	}

	query := mux.Vars(r)["type"]

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

	var functionType *pbtypes.FunctionType
	if functionType, r.URL, err = p.queryURL(query, tgt.GetId()); err != nil {
		p.log.Error(err.Error())
		return
	}

	tgt.Namespace = functionType.GetNamespace()
	tgt.Type = functionType.GetType()

	if body, err = proto.Marshal(&toFunction); err != nil {
		p.log.Error(err.Error())
		return
	}

	r.Body = io.NopCloser(bytes.NewBuffer(body))
	r.ContentLength = int64(len(body))

	r.Header.Add(forwardedHeader, r.Host)
	r.Host = r.URL.Host

	p.log.Debugf("director: (%+v) uri: %s", r.Header, r.RequestURI)
}

func (p *Proxy) modifyResponse(r *http.Response) (err error) {
	p.log.Debugf("modify: (%s)", r.Status)

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

	p.log.Debugf("modify: ContentLength (%d)", r.ContentLength)

	return
}

func (p *Proxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	p.log.Debugf("error handler: (%s)", r.Method)

	if err != nil {
		p.log.Error(err)
	}

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
func (p *Proxy) queryURL(query, id string) (functionType *pbtypes.FunctionType, u *url.URL, err error) {
	p.log.Debugf("qdsl: (%s)", query)

	// read meta from func object
	nodes, err := qdsl.Qdsl(p.ctx, query, qdsl.WithObject(), qdsl.WithId(), qdsl.WithLink())
	if err != nil {
		return
	}

	if len(nodes) == 0 {
		err = fmt.Errorf("unknown function declaration: %s", query)
		return
	}

	if len(nodes) > 1 {
		err = fmt.Errorf("multiple declaration of function: %s", query)
		return
	}

	var function pbtypes.Function
	if err = json.Unmarshal(nodes[0].Object, &function); err != nil {
		return
	}

	functionType = function.GetFunctionType()

	// one instance of function
	if !function.Grounded {
		var link pbtypes.FunctionRoute
		if err = json.Unmarshal(nodes[0].Link, &link); err != nil {
			return
		}

		u, err = url.Parse(link.Url)
		return
	}

	p.log.Debugf("finder: (%s -> %s)", nodes[0].Id, id)

	var name string
	resp, err := finder.Links(p.ctx, nodes[0].Id.String(), id, name)
	if err != nil {
		return
	}

	if len(resp) == 0 {
		err = fmt.Errorf("unknown function implementation: %s -> %s", query, id)
		return
	}

	if len(resp) > 1 {
		err = fmt.Errorf("multiple implementation of function: %s -> %s", functionType, id)
		return
	}

	var link pbtypes.FunctionRoute
	if err = json.Unmarshal(resp[0].GetPayload(), &link); err != nil {
		return
	}

	u, err = url.Parse(link.Url)
	return
}

func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		p.ServeHTTP(w, r)
	}
}
