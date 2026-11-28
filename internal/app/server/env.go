package server

import (
	"log/slog"
	"net"

	envv1 "github.com/Skoowshot/qrix-runtime/gen/go/env/v1"
	"github.com/Skoowshot/qrix-runtime/internal/app/grpchandler"
	"github.com/Skoowshot/qrix-runtime/internal/app/middleware"
	"github.com/Skoowshot/qrix-runtime/internal/app/service"
	"google.golang.org/grpc"
)

type Environment struct {
	listener   net.Listener
	grpcServer *grpc.Server
}

func ListenEnvironment(addr string, envService *service.Environment) (*Environment, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		middleware.Error(),
	), grpc.ChainStreamInterceptor(
		middleware.StreamError(),
		middleware.StreamSessionToken(),
		middleware.StreamWorkspace(),
	))

	envv1.RegisterEnvironmentServiceServer(grpcServer, grpchandler.NewEnvironment(envService))

	return &Environment{
		listener:   lis,
		grpcServer: grpcServer,
	}, nil
}

func (c *Environment) Start() error {
	slog.Info("starting env server", "addr", c.listener.Addr())
	return c.grpcServer.Serve(c.listener)
}

func (c *Environment) GracefulStop() {
	slog.Info("gracefully stopping env server")
	c.grpcServer.GracefulStop()
}

func (c *Environment) Stop() {
	slog.Info("stopping env server")
	c.grpcServer.Stop()
}
