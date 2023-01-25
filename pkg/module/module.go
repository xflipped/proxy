// Copyright 2022 Listware

package module

import (
	"git.fg-tech.ru/listware/go-core/pkg/module"
)

func New(namespace string, opts ...module.Opt) module.Module {
	return module.New(namespace, opts...)
}
