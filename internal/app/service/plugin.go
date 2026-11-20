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

// InstantiateSlots resolves, creates, and puts plugin slots into the specified workspace.
// If any step fails, created slots are closed and any put slots from the batch are rolled back.
func (s *Plugin) InstantiateSlots(ctx context.Context, wsID workspace.ID, descriptors []SlotDescriptor) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ws, ok := s.wsRepo.Get(wsID)
	if !ok {
		return fmt.Errorf("%w: %d", workspace.ErrNotFound, wsID)
	}

	plugins, configs, err := resolveDescriptors(ctx, s.pluginRepo, descriptors)
	if err != nil {
		return err
	}

	slots, err := createSlots(ctx, configs, plugins)
	if err != nil {
		return err
	}

	if err := ws.PutSlots(slots); err != nil {
		closeSlots(slots)
		return err
	}
	return nil
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

	var wg sync.WaitGroup
	var errs []error
	var mu sync.Mutex

	plugins := make(map[plugin.ID]struct{})
	for _, id := range pluginIDs {
		if _, exists := plugins[id]; exists {
			continue
		}
		plugins[id] = struct{}{}

		slot := ws.RemoveSlot(id)
		if slot == nil {
			continue
		}

		wg.Go(func() {
			err := slot.Close()
			if err != nil {
				mu.Lock()
				defer mu.Unlock()

				errs = append(errs, err)
				return
			}
		})
	}
	wg.Wait()

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

func resolveDescriptors(ctx context.Context, repo *plugin.Registry, descriptors []SlotDescriptor) (map[plugin.ID]plugin.Plugin, map[plugin.ID]plugin.Config, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	var errs []error

	plugins := make(map[plugin.ID]plugin.Plugin)
	configs := make(map[plugin.ID]plugin.Config)
	for _, desc := range descriptors {
		id := desc.ID
		plg, ok := repo.Get(id)
		if !ok {
			errs = append(errs, fmt.Errorf("%w: %d", plugin.ErrNotFound, id))
		}
		plugins[id] = plg
		configs[id] = desc.Config
	}

	return plugins, configs, errors.Join(errs...)
}

func createSlots(ctx context.Context, descriptors map[plugin.ID]plugin.Config, plugins map[plugin.ID]plugin.Plugin) (map[plugin.ID]*plugin.Slot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	var errs []error

	slots := make(map[plugin.ID]*plugin.Slot)
	for id, config := range descriptors {
		plg, ok := plugins[id]
		if !ok {
			mu.Lock()
			errs = append(errs, fmt.Errorf("%w: %d", plugin.ErrNotFound, id))
			mu.Unlock()
			continue
		}

		wg.Go(func() {
			slot, err := createSlot(ctx, config, plg)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			slots[id] = slot
		})
	}
	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		closeSlots(slots)
		return nil, err
	}
	return slots, nil
}

func createSlot(ctx context.Context, config plugin.Config, plg plugin.Plugin) (*plugin.Slot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	slot := plugin.NewSlot()
	init := plugin.InitConfig{
		Config: config,
	}

	if err := plg.Init(ctx, init, slot); err != nil {
		_ = slot.Close()
		return nil, err
	}
	return slot, nil
}

func closeSlots(slots map[plugin.ID]*plugin.Slot) {
	for _, slot := range slots {
		if slot == nil {
			continue
		}
		_ = slot.Close()
	}
}
