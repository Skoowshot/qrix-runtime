package grpchandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	pluginv1 "github.com/Skoowshot/qrix-runtime/gen/go/plugin/v1"
	"github.com/Skoowshot/qrix-runtime/internal/app/middleware"
	"github.com/Skoowshot/qrix-runtime/internal/app/service"
	"github.com/Skoowshot/qrix-runtime/internal/plugin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var emptyConfig = json.RawMessage("{}")

// Plugin implements the gRPC server for managing plugin slots and listing registered plugins.
type Plugin struct {
	service *service.Plugin
	pluginv1.UnimplementedPluginServiceServer
}

// NewPlugin creates a new gRPC Plugin handler backed by the given service.
func NewPlugin(service *service.Plugin) *Plugin {
	return &Plugin{
		service: service,
	}
}

// InstantiateSlots initializes and attaches plugin slots to the workspace specified in context.
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
		return nil, err
	}

	return &pluginv1.InstantiateSlotsResponse{}, nil
}

// ListSlots returns the IDs of all active plugin slots in the workspace specified in context.
func (p *Plugin) ListSlots(ctx context.Context, req *pluginv1.ListSlotsRequest) (*pluginv1.ListSlotsResponse, error) {
	wsID, ok := middleware.WorkspaceFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "missing workspace")
	}

	ids, err := p.service.ListSlots(ctx, wsID)
	if err != nil {
		return nil, err
	}

	mappedIds := make([]uint64, len(ids))
	for i, id := range ids {
		mappedIds[i] = uint64(id)
	}

	return &pluginv1.ListSlotsResponse{
		PluginIds: mappedIds,
	}, nil
}

// DestroySlots detaches and closes the specified plugin slots from the workspace in context.
func (p *Plugin) DestroySlots(ctx context.Context, req *pluginv1.DestroySlotsRequest) (*pluginv1.DestroySlotsResponse, error) {
	wsID, ok := middleware.WorkspaceFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "missing workspace")
	}

	pluginIds := make([]plugin.ID, len(req.PluginIds))
	for i, id := range req.PluginIds {
		pluginIds[i] = plugin.ID(id)
	}

	if err := p.service.DestroySlots(ctx, wsID, pluginIds); err != nil {
		return nil, err
	}

	return &pluginv1.DestroySlotsResponse{}, nil
}

// ListPlugins returns all registered plugin IDs and their name-to-ID mapping.
func (p *Plugin) ListPlugins(ctx context.Context, req *pluginv1.ListPluginsRequest) (*pluginv1.ListPluginsResponse, error) {
	ids, mapNames, err := p.service.ListPlugins(ctx)
	if err != nil {
		return nil, err
	}

	mappedIds := make([]uint64, len(ids))
	mappedMapNames := make(map[string]uint64)

	for i, id := range ids {
		mappedIds[i] = uint64(id)
	}
	for name, id := range mapNames {
		mappedMapNames[string(name)] = uint64(id)
	}

	return &pluginv1.ListPluginsResponse{
		Ids:      mappedIds,
		NameToId: mappedMapNames,
	}, nil
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
