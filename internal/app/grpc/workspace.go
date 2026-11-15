package grpc

import (
	"context"
	"errors"
	"log/slog"

	workspacev1 "github.com/Skoowshot/qrix-runtime/gen/workspace/v1"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type WorkspaceHandler struct {
	service *workspace.Service
	workspacev1.UnimplementedWorkspaceServiceServer
}

func NewWorkspaceHandler(service *workspace.Service) *WorkspaceHandler {
	return &WorkspaceHandler{
		service: service,
	}
}

func (h *WorkspaceHandler) Create(
	ctx context.Context,
	req *workspacev1.CreateRequest,
) (*workspacev1.CreateResponse, error) {
	name := workspace.Name(req.Name)
	id, err := h.service.Create(ctx, name)
	if err != nil {
		if errors.Is(err, workspace.ErrNameAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}

		slog.ErrorContext(ctx, "failed to create workspace", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &workspacev1.CreateResponse{
		Id: uint64(id),
	}, nil
}
func (h *WorkspaceHandler) GetByName(
	ctx context.Context,
	req *workspacev1.GetByNameRequest,
) (*workspacev1.GetByNameResponse, error) {
	name := workspace.Name(req.Name)
	id, err := h.service.GetByName(ctx, name)
	if err != nil {
		if errors.Is(err, workspace.ErrNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		slog.ErrorContext(ctx, "failed to get workspace by name", "error", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &workspacev1.GetByNameResponse{
		Id: uint64(id),
	}, nil
}
