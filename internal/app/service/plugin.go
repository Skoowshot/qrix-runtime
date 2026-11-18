package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Skoowshot/qrix-runtime/internal/plugin"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
	"golang.org/x/sync/errgroup"
)

// SlotDescriptor pairs a plugin ID with its configuration for slot instantiation.
type SlotDescriptor struct {
	ID     plugin.ID
	Config plugin.Config
}

// Plugin manages plugin slot lifecycles within workspaces and queries registered plugins.
type Plugin struct {
	pluginRepo *plugin.Registry
	wsRepo     *workspace.Registry
}

// NewPlugin creates a new Plugin service backed by the given plugin and workspace registries.
func NewPlugin(pluginRepo *plugin.Registry, wsRepo *workspace.Registry) *Plugin {
	return &Plugin{
		pluginRepo: pluginRepo,
		wsRepo:     wsRepo,
	}
}

// InstantiateSlots resolves, initializes, and attaches plugin slots to the specified workspace.
// If any step fails, initialized slots are closed and any attached slots from the batch are rolled back.
func (s *Plugin) InstantiateSlots(ctx context.Context, wsID workspace.ID, descriptors []SlotDescriptor) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ws, ok := s.wsRepo.Get(wsID)
	if !ok {
		return fmt.Errorf("%w: %d", workspace.ErrNotFound, wsID)
	}

	plugins, err := s.resolvePlugins(ws, descriptors)
	if err != nil {
		return err
	}

	slots, err := s.initSlots(ctx, descriptors, plugins)
	if err != nil {
		return err
	}

	return attachSlots(ws, descriptors, slots)
}

// ListSlots returns the IDs of all plugin slots currently attached to the specified workspace.
func (s *Plugin) ListSlots(ctx context.Context, wsID workspace.ID) ([]plugin.ID, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ws, ok := s.wsRepo.Get(wsID)
	if !ok {
		return nil, fmt.Errorf("%w: %d", workspace.ErrNotFound, wsID)
	}

	return ws.ListSlots(), nil
}

// DestroySlots detaches and closes the specified plugin slots in parallel, joining any close errors.
func (s *Plugin) DestroySlots(ctx context.Context, wsID workspace.ID, pluginIDs []plugin.ID) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	ws, ok := s.wsRepo.Get(wsID)
	if !ok {
		return fmt.Errorf("%w: %d", workspace.ErrNotFound, wsID)
	}

	var g errgroup.Group
	errs := make([]error, len(pluginIDs))

	for i, id := range pluginIDs {
		g.Go(destroySlotGoroutine(ws, id, errs, i))
	}

	_ = g.Wait()
	return errors.Join(errs...)
}

// ListPlugins returns all registered plugin IDs and their name-to-ID mapping.
func (s *Plugin) ListPlugins(ctx context.Context) ([]plugin.ID, map[plugin.Name]plugin.ID, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	ids := s.pluginRepo.List()
	mapNames := s.pluginRepo.MapNames()

	return ids, mapNames, nil
}

func (s *Plugin) resolvePlugins(ws *workspace.Workspace, descriptors []SlotDescriptor) ([]plugin.Plugin, error) {
	seen := make(map[plugin.ID]struct{}, len(descriptors))
	plugins := make([]plugin.Plugin, len(descriptors))
	var errs []error

	for i, desc := range descriptors {
		if _, dup := seen[desc.ID]; dup {
			errs = append(errs, fmt.Errorf("%w: %v", plugin.ErrSlotAlreadyExists, desc.ID))
			continue
		}
		seen[desc.ID] = struct{}{}

		plg, ok := s.pluginRepo.Get(desc.ID)
		if !ok {
			errs = append(errs, fmt.Errorf("%w: %v", plugin.ErrNotFound, desc.ID))
			continue
		}
		if _, ok := ws.Slot(desc.ID); ok {
			errs = append(errs, fmt.Errorf("%w: %v", plugin.ErrSlotAlreadyExists, desc.ID))
			continue
		}
		plugins[i] = plg
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return plugins, nil
}

func (s *Plugin) initSlots(ctx context.Context, descriptors []SlotDescriptor, plugins []plugin.Plugin) ([]*plugin.Slot, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var g errgroup.Group
	slots := make([]*plugin.Slot, len(descriptors))
	errs := make([]error, len(descriptors))

	for i, desc := range descriptors {
		g.Go(initSlotGoroutine(ctx, cancel, desc, plugins, i, slots, errs))
	}

	_ = g.Wait()
	if err := errors.Join(errs...); err != nil {
		closeSlots(slots)
		return nil, err
	}
	return slots, nil
}

func initSlotGoroutine(
	ctx context.Context,
	cancel context.CancelFunc,
	desc SlotDescriptor,
	plugins []plugin.Plugin,
	idx int,
	slots []*plugin.Slot,
	errs []error,
) func() error {
	return func() error {
		slot := plugin.NewSlot()
		initCfg := plugin.InitConfig{
			Config: desc.Config,
		}

		if err := plugins[idx].Init(ctx, initCfg, slot); err != nil {
			_ = slot.Close()
			errs[idx] = fmt.Errorf("init plugin %v: %w", desc.ID, err)
			cancel()
			return nil
		}
		slots[idx] = slot
		return nil
	}
}
func destroySlotGoroutine(ws *workspace.Workspace, id plugin.ID, errs []error, idx int) func() error {
	return func() error {
		slot := ws.RemoveSlot(id)
		if slot == nil {
			return nil
		}
		if err := slot.Close(); err != nil {
			errs[idx] = fmt.Errorf("close slot %d: %w", id, err)
		}
		return nil
	}
}

func attachSlots(ws *workspace.Workspace, descriptors []SlotDescriptor, slots []*plugin.Slot) error {
	for i, desc := range descriptors {
		if ws.PutSlot(desc.ID, slots[i]) {
			continue
		}
		closeSlots(slots[i:])
		removeSlots(ws, descriptors[:i])
		return fmt.Errorf("%w: %v", plugin.ErrSlotAlreadyExists, desc.ID)
	}
	return nil
}

func removeSlots(ws *workspace.Workspace, descriptors []SlotDescriptor) {
	for _, j := range descriptors {
		removed := ws.RemoveSlot(j.ID)
		if removed == nil {
			continue
		}
		_ = removed.Close()
	}
}

func closeSlots(slots []*plugin.Slot) {
	for _, slot := range slots {
		if slot == nil {
			continue
		}
		_ = slot.Close()
	}
}
