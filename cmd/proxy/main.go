// Copyright 2022 Listware

package main

import (
	"fmt"
	"os"

	"git.fg-tech.ru/listware/proxy/pkg/agent"
)

func main() {
	if err := agent.Proxy.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
