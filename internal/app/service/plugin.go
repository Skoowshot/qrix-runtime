package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Skoowshot/qrix-runtime/internal/plugin"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
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
func (s *Plugin) DestroySlots(ctx context.Context, wsID workspace.ID, pluginIDs map[plugin.ID]bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	ws, ok := s.wsRepo.Get(wsID)
	if !ok {
		return fmt.Errorf("%w: %d", workspace.ErrNotFound, wsID)
	}

	var g sync.WaitGroup
	errs := make([]error, len(pluginIDs))

	i := 0
	for id := range pluginIDs {
		g.Go(func() {
			if err := destroySlot(ws, id); err != nil {
				errs[i] = err
			}
		})
		i += 1
	}

	g.Wait()
	return errors.Join(errs...)
}

// ListPlugins returns all registered plugin IDs and their name-to-ID mapping.
func (s *Plugin) ListPlugins(ctx context.Context) ([]plugin.ID, map[plugin.Name]plugin.ID, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	ids, mapNames := s.pluginRepo.Snapshot()
	return ids, mapNames, nil
}

func (s *Plugin) resolvePlugins(ws *workspace.Workspace, descriptors []SlotDescriptor) ([]plugin.Plugin, error) {
	seen := make(map[plugin.ID]bool, len(descriptors))
	plugins := make([]plugin.Plugin, len(descriptors))
	var errs []error

	for i, desc := range descriptors {
		if _, dup := seen[desc.ID]; dup {
			errs = append(errs, fmt.Errorf("%w: %v", plugin.ErrSlotAlreadyExists, desc.ID))
			continue
		}
		seen[desc.ID] = true

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

	var g sync.WaitGroup
	slots := make([]*plugin.Slot, len(descriptors))
	errs := make([]error, len(descriptors))

	for i, desc := range descriptors {
		g.Go(func() {
			if err := initSlot(ctx, cancel, desc, plugins, slots, i); err != nil {
				errs[i] = err
			}
		})
	}

	g.Wait()
	if err := errors.Join(errs...); err != nil {
		closeSlots(slots)
		return nil, err
	}
	return slots, nil
}

func initSlot(
	ctx context.Context,
	cancel context.CancelFunc,
	desc SlotDescriptor,
	plugins []plugin.Plugin,
	slots []*plugin.Slot,
	idx int,
) error {
	slot := plugin.NewSlot()
	initCfg := plugin.InitConfig{
		Config: desc.Config,
	}

	if err := plugins[idx].Init(ctx, initCfg, slot); err != nil {
		_ = slot.Close()
		cancel()
		return fmt.Errorf("init plugin %v: %w", desc.ID, err)
	}

	slots[idx] = slot
	return nil
}

func destroySlot(ws *workspace.Workspace, id plugin.ID) error {
	slot, ok := ws.Slot(id)
	if !ok {
		return nil
	}
	if ws.RemoveSlot(id, slot) {
		return nil
	}
	if err := slot.Close(); err != nil {
		return fmt.Errorf("close slot %d: %w", id, err)
	}

	return nil
}

func attachSlots(ws *workspace.Workspace, descriptors []SlotDescriptor, slots []*plugin.Slot) error {
	for i, desc := range descriptors {
		if ws.PutSlot(desc.ID, slots[i]) {
			continue
		}
		closeSlots(slots[i:])
		removeSlots(ws, descriptors[:i], slots[:i])
		return fmt.Errorf("%w: %v", plugin.ErrSlotAlreadyExists, desc.ID)
	}
	return nil
}

func removeSlots(ws *workspace.Workspace, descriptors []SlotDescriptor, slots []*plugin.Slot) {
	for i, desc := range descriptors {
		if ws.RemoveSlot(desc.ID, slots[i]) {
			_ = slots[i].Close()
		}
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
