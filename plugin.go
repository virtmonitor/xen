package main

import (
	"os"

	"github.com/hashicorp/go-hclog"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/virtmonitor/plugins"
	xen "github.com/virtmonitor/xen/driver"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "XEN",
		Output: os.Stderr,
	})

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugins.Handshake,
		Plugins: map[string]plugin.Plugin{
			"driver_grpc": &plugins.DriverGrpcPlugin{Impl: &xen.Xen{}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
		Logger:     logger,
	})
}
