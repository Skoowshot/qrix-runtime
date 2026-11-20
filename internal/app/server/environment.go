package server

import (
	"log/slog"
	"net"

	"google.golang.org/grpc"
)

type Environment struct {
	listener   net.Listener
	grpcServer *grpc.Server
}

func ListenEnvironment(addr string) (*Environment, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	grpcServer := grpc.NewServer()
	return &Environment{
		listener:   lis,
		grpcServer: grpcServer,
	}, nil
}

func (c *Environment) Start() error {
	slog.Info("starting environment server", "addr", c.listener.Addr())
	return c.grpcServer.Serve(c.listener)
}

func (c *Environment) GracefulStop() {
	slog.Info("gracefully stopping environment server")
	c.grpcServer.GracefulStop()
}

func (c *Environment) Stop() {
	slog.Info("stopping environment server")
	c.grpcServer.Stop()
}
