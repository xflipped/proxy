module git.fg-tech.ru/listware/proxy

go 1.19

replace git.fg-tech.ru/listware/proto => github.com/foliagecp/proto v0.1.3

replace git.fg-tech.ru/listware/cmdb => github.com/foliagecp/cmdb v0.1.4

require (
	git.fg-tech.ru/listware/cmdb v0.1.4
	git.fg-tech.ru/listware/proto v0.1.3
	github.com/gorilla/mux v1.8.0
	github.com/urfave/cli/v2 v2.24.2
	go.uber.org/zap v1.24.0
	google.golang.org/protobuf v1.28.1
)

require (
	github.com/arangodb/go-driver v1.4.1 // indirect
	github.com/arangodb/go-velocypack v0.0.0-20200318135517-5af53c29c67e // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.5.0 // indirect
	golang.org/x/sys v0.4.0 // indirect
	golang.org/x/text v0.6.0 // indirect
	google.golang.org/genproto v0.0.0-20221118155620-16455021b5e6 // indirect
	google.golang.org/grpc v1.52.1 // indirect
)
