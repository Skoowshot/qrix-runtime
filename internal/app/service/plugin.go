package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Skoowshot/qrix-runtime/internal/plugin"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
	"golang.org/x/sync/errgroup"
)

type SlotDescriptor struct {
	ID     plugin.ID
	Config plugin.Config
}

type Plugin struct {
	pluginRepo *plugin.Registry
	wsRepo     *workspace.Registry
}

func NewPlugin(pluginRepo *plugin.Registry, wsRepo *workspace.Registry) *Plugin {
	return &Plugin{
		pluginRepo: pluginRepo,
		wsRepo:     wsRepo,
	}
}

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
	g, gCtx := errgroup.WithContext(ctx)
	slots := make([]*plugin.Slot, len(descriptors))

	for i, desc := range descriptors {
		g.Go(initSlotGoroutine(gCtx, desc, i, plugins, slots))
	}

	if err := g.Wait(); err != nil {
		closeSlots(slots)
		return nil, err
	}
	return slots, nil
}

func initSlotGoroutine(ctx context.Context, desc SlotDescriptor, idx int, plugins []plugin.Plugin, slots []*plugin.Slot) func() error {
	return func() error {
		slot := plugin.NewSlot()
		initCfg := plugin.InitConfig{
			Config: desc.Config,
		}

		if err := plugins[idx].Init(ctx, initCfg, slot); err != nil {
			_ = slot.Close()
			return fmt.Errorf("init plugin %v: %w", desc.ID, err)
		}
		slots[idx] = slot
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
