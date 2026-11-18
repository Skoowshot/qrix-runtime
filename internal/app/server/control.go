package server

import (
	"log/slog"
	"net"

	pluginv1 "github.com/Skoowshot/qrix-runtime/gen/go/plugin/v1"
	workspacev1 "github.com/Skoowshot/qrix-runtime/gen/go/workspace/v1"
	"github.com/Skoowshot/qrix-runtime/internal/app/grpchandler"
	"github.com/Skoowshot/qrix-runtime/internal/app/middleware"
	"github.com/Skoowshot/qrix-runtime/internal/app/service"
	"google.golang.org/grpc"
)

type Control struct {
	listener   net.Listener
	grpcServer *grpc.Server
}

func ListenControl(
	addr string,
	wsService *service.Workspace,
	plgService *service.Plugin,
) (*Control, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		middleware.Workspace(),
	))

	workspacev1.RegisterWorkspaceServiceServer(grpcServer, grpchandler.NewWorkspace(wsService))
	pluginv1.RegisterPluginServiceServer(grpcServer, grpchandler.NewPlugin(plgService))

	return &Control{
		listener:   lis,
		grpcServer: grpcServer,
	}, nil
}

func (c *Control) Start() error {
	slog.Info("starting control server", "addr", c.listener.Addr())
	return c.grpcServer.Serve(c.listener)
}

func (c *Control) GracefulStop() {
	slog.Info("gracefully stopping control server")
	c.grpcServer.GracefulStop()
}

func (c *Control) Stop() {
	slog.Info("stopping control server")
	c.grpcServer.Stop()
}
