// Copyright 2022 Listware

package proxy

import (
	"go.uber.org/zap"
)

type Opt func(*Proxy) error

func WithPort(port int) Opt {
	return func(p *Proxy) (err error) {
		p.port = port
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
