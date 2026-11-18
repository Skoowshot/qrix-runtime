package grpchandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	pluginv1 "github.com/Skoowshot/qrix-runtime/gen/go/plugin/v1"
	"github.com/Skoowshot/qrix-runtime/internal/app/middleware"
	"github.com/Skoowshot/qrix-runtime/internal/app/service"
	"github.com/Skoowshot/qrix-runtime/internal/plugin"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var emptyConfig = json.RawMessage("{}")

type Plugin struct {
	service *service.Plugin
	pluginv1.UnimplementedPluginServiceServer
}

func NewPlugin(service *service.Plugin) *Plugin {
	return &Plugin{
		service: service,
	}
}

func (p *Plugin) InstantiateSlots(ctx context.Context, req *pluginv1.InstantiateSlotsRequest) (*pluginv1.InstantiateSlotsResponse, error) {
	wsID, ok := middleware.WorkspaceFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "missing workspace")
	}

	descriptors, err := mapDescriptors(req.Descriptors)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid descriptors: %v", err)
	}
	if err := p.service.InstantiateSlots(ctx, wsID, descriptors); err != nil {
		switch {
		case errors.Is(err, workspace.ErrNotFound),
			errors.Is(err, plugin.ErrNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, plugin.ErrSlotAlreadyExists):
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		slog.ErrorContext(ctx, "failed to instantiate slots", "err", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &pluginv1.InstantiateSlotsResponse{}, nil
}
func (p *Plugin) ListSlots(ctx context.Context, req *pluginv1.ListSlotsRequest) (*pluginv1.ListSlotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method ListSlots not implemented")
}
func (p *Plugin) DestroySlots(ctx context.Context, req *pluginv1.DestroySlotsRequest) (*pluginv1.DestroySlotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method DestroySlots not implemented")
}
func (p *Plugin) ListPlugins(ctx context.Context, req *pluginv1.ListPluginsRequest) (*pluginv1.ListPluginsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method ListPlugins not implemented")
}

func mapDescriptors(descriptors []*pluginv1.SlotDescriptor) ([]service.SlotDescriptor, error) {
	result := make([]service.SlotDescriptor, len(descriptors))
	errs := make([]error, 0, len(descriptors))
	for i, desc := range descriptors {
		if desc == nil {
			errs = append(errs, fmt.Errorf("index %d: nil descriptor", i))
			continue
		}

		id := plugin.ID(desc.PluginId)
		config, err := mapConfig(desc.Config)
		if err != nil {
			errs = append(errs, fmt.Errorf("plugin %d: %w", id, err))
			continue
		}

		result[i] = service.SlotDescriptor{
			ID:     id,
			Config: config,
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return result, nil
}

func mapConfig(cfg []byte) (plugin.Config, error) {
	if len(cfg) == 0 {
		return emptyConfig, nil
	}
	config := json.RawMessage(cfg)
	if !config.IsValid() {
		return nil, fmt.Errorf("invalid config")
	}
	return config, nil
}
