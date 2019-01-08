package main

import (
	plugin "github.com/hashicorp/go-plugin"
	"github.com/virtmonitor/plugins"
	xen "github.com/virtmonitor/xen/driver"
)

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugins.Handshake,
		Plugins: map[string]plugin.Plugin{
			"driver_grpc": &plugins.DriverGrpcPlugin{Impl: &xen.Xen{}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
