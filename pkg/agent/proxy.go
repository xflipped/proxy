// Copyright 2022 Listware

package agent

import (
	"git.fg-tech.ru/listware/proxy/pkg/proxy"
	"github.com/urfave/cli/v2"
)

var Proxy = &cli.App{
	Name:    "statefun-proxy",
	Usage:   "Flink's Stateful Functions Proxy",
	Version: "v0.0.1",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "listen",
			Aliases: []string{"l"},
			Usage:   "Address to listen on",
			Value:   ":80",
			EnvVars: []string{"STATEFUN_PROXY_ADDR"},
		},
		&cli.BoolFlag{
			Name:    "debug",
			Aliases: []string{"d"},
			Usage:   "Debug log level",
			Value:   false,
			EnvVars: []string{"STATEFUN_PROXY_DEBUG"},
		},
	},
	Action: func(ctx *cli.Context) (err error) {
		p, err := proxy.New(proxy.WithAddr(ctx.String("listen")))
		if err != nil {
			return
		}
		if ctx.Bool("debug") {
			if err = p.Configure(proxy.WithDebug()); err != nil {
				return
			}
		}
		return p.Run(ctx.Context)
	},
}
