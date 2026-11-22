package grpchandler

import (
	"context"

	envv1 "github.com/Skoowshot/qrix-runtime/gen/go/env/v1"
	"github.com/Skoowshot/qrix-runtime/internal/app/service"
	"github.com/Skoowshot/qrix-runtime/internal/env"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Environment struct {
	service *service.Environment
	envv1.UnimplementedEnvironmentServiceServer
}

func NewEnvironment(service *service.Environment) *Environment {
	return &Environment{
		service: service,
	}
}

func (e *Environment) Connect(ctx context.Context, req *envv1.ConnectRequest) (*envv1.ConnectResponse, error) {
	wsID := workspace.ID(req.WorkspaceId)
	name := env.Name(req.Name)
	metadata := env.Metadata(req.Metadata)

	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "missing name")
	}
	if len(metadata) == 0 {
		metadata = env.Metadata("{}")
	}
	if !metadata.IsValid() {
		return nil, status.Error(codes.InvalidArgument, "invalid metadata")
	}

	token, err := e.service.Connect(ctx, wsID, name, metadata)
	if err != nil {
		return nil, err
	}

	return &envv1.ConnectResponse{
		SessionToken: string(token),
	}, nil
}
