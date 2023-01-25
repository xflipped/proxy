// Copyright 2022 Listware

package proxy

import (
	"go.uber.org/zap"
)

type Opt func(*Proxy) error

func WithAddr(addr string) Opt {
	return func(p *Proxy) (err error) {
		p.addr = addr
		return
	}
}

func WithDebug() Opt {
	return func(p *Proxy) (err error) {
		l, err := zap.NewDevelopment()
		if err != nil {
			return
		}
		p.log = l.Sugar()
		return
	}
}
