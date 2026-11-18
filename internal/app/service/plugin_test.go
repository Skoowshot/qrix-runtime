package service

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/Skoowshot/qrix-runtime/internal/plugin"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
)

func TestPluginService_InstantiateSlots(t *testing.T) {
	errInit := errors.New("init failed")

	tests := []struct {
		name        string
		setup       func(t *testing.T, svc *Plugin, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []SlotDescriptor)
		ctx         func() context.Context
		expectedErr error
		verify      func(t *testing.T, svc *Plugin, wsID workspace.ID)
	}{
		{
			name: "instantiates slots successfully",
			setup: func(t *testing.T, _ *Plugin, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []SlotDescriptor) {
				wsID := mustCreateWorkspace(t, wsRepo, "alpha")
				id1 := mustRegisterPlugin(t, plgRepo, "p1", &stubPlugin{})
				id2 := mustRegisterPlugin(t, plgRepo, "p2", &stubPlugin{})
				return wsID, []SlotDescriptor{
					{ID: id1, Config: plugin.Config(`{"a":1}`)},
					{ID: id2, Config: plugin.Config(`{}`)},
				}
			},
			ctx:         context.Background,
			expectedErr: nil,
			verify: func(t *testing.T, svc *Plugin, wsID workspace.ID) {
				got, err := svc.ListSlots(context.Background(), wsID)
				if err != nil {
					t.Fatalf("ListSlots() error = %v", err)
				}
				if len(got) != 2 {
					t.Errorf("len(ListSlots()) = %d, want 2", len(got))
				}
			},
		},
		{
			name: "missing workspace returns NotFound",
			setup: func(t *testing.T, _ *Plugin, plgRepo *plugin.Registry, _ *workspace.Registry) (workspace.ID, []SlotDescriptor) {
				id1 := mustRegisterPlugin(t, plgRepo, "p1", &stubPlugin{})
				return 999, []SlotDescriptor{{ID: id1}}
			},
			ctx:         context.Background,
			expectedErr: workspace.ErrNotFound,
		},
		{
			name: "missing plugin returns NotFound",
			setup: func(t *testing.T, _ *Plugin, _ *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []SlotDescriptor) {
				wsID := mustCreateWorkspace(t, wsRepo, "alpha")
				return wsID, []SlotDescriptor{{ID: 999}}
			},
			ctx:         context.Background,
			expectedErr: plugin.ErrNotFound,
		},
		{
			name: "duplicate plugin IDs in request returns ErrSlotAlreadyExists",
			setup: func(t *testing.T, _ *Plugin, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []SlotDescriptor) {
				wsID := mustCreateWorkspace(t, wsRepo, "alpha")
				id1 := mustRegisterPlugin(t, plgRepo, "p1", &stubPlugin{})
				return wsID, []SlotDescriptor{{ID: id1}, {ID: id1}}
			},
			ctx:         context.Background,
			expectedErr: plugin.ErrSlotAlreadyExists,
		},
		{
			name: "already instantiated slot returns ErrSlotAlreadyExists",
			setup: func(t *testing.T, svc *Plugin, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []SlotDescriptor) {
				wsID := mustCreateWorkspace(t, wsRepo, "alpha")
				id1 := mustRegisterPlugin(t, plgRepo, "p1", &stubPlugin{})
				if err := svc.InstantiateSlots(context.Background(), wsID, []SlotDescriptor{{ID: id1}}); err != nil {
					t.Fatalf("setup InstantiateSlots() error = %v", err)
				}
				return wsID, []SlotDescriptor{{ID: id1}}
			},
			ctx:         context.Background,
			expectedErr: plugin.ErrSlotAlreadyExists,
		},
		{
			name: "init failure closes initialized slots and returns error",
			setup: func(t *testing.T, _ *Plugin, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []SlotDescriptor) {
				wsID := mustCreateWorkspace(t, wsRepo, "alpha")
				closer := &stubCloser{}
				id1 := mustRegisterPlugin(t, plgRepo, "p1", &stubPlugin{
					initFn: func(_ context.Context, _ plugin.InitConfig, slot *plugin.Slot) error {
						slot.TrackCloser(closer)
						return nil
					},
				})
				id2 := mustRegisterPlugin(t, plgRepo, "p2", &stubPlugin{
					initFn: func(_ context.Context, _ plugin.InitConfig, _ *plugin.Slot) error {
						return errInit
					},
				})
				t.Cleanup(func() {
					if closer.closed.Load() != 1 {
						t.Errorf("closer.closed = %d, want 1", closer.closed.Load())
					}
				})
				return wsID, []SlotDescriptor{{ID: id1}, {ID: id2}}
			},
			ctx:         context.Background,
			expectedErr: errInit,
			verify: func(t *testing.T, svc *Plugin, wsID workspace.ID) {
				got, err := svc.ListSlots(context.Background(), wsID)
				if err != nil {
					t.Fatalf("ListSlots() error = %v", err)
				}
				if len(got) != 0 {
					t.Errorf("len(ListSlots()) = %d, want 0", len(got))
				}
			},
		},
		{
			name: "canceled context returns error",
			setup: func(t *testing.T, _ *Plugin, _ *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []SlotDescriptor) {
				wsID := mustCreateWorkspace(t, wsRepo, "alpha")
				return wsID, nil
			},
			ctx:         canceledContext,
			expectedErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plgRepo := plugin.NewRegistry()
			wsRepo := workspace.NewRegistry()
			svc := NewPlugin(plgRepo, wsRepo)

			wsID, descs := tt.setup(t, svc, plgRepo, wsRepo)
			err := svc.InstantiateSlots(tt.ctx(), wsID, descs)
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("error = %v, want %v", err, tt.expectedErr)
			}
			if tt.verify != nil {
				tt.verify(t, svc, wsID)
			}
		})
	}
}

func TestPluginService_ListSlots(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, svc *Plugin, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []plugin.ID)
		ctx         func() context.Context
		expectedErr error
	}{
		{
			name: "empty workspace returns no slots",
			setup: func(t *testing.T, _ *Plugin, _ *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []plugin.ID) {
				return mustCreateWorkspace(t, wsRepo, "alpha"), nil
			},
			ctx:         context.Background,
			expectedErr: nil,
		},
		{
			name: "returns instantiated slot IDs",
			setup: func(t *testing.T, svc *Plugin, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []plugin.ID) {
				wsID := mustCreateWorkspace(t, wsRepo, "alpha")
				id1 := mustRegisterPlugin(t, plgRepo, "p1", &stubPlugin{})
				id2 := mustRegisterPlugin(t, plgRepo, "p2", &stubPlugin{})
				if err := svc.InstantiateSlots(context.Background(), wsID, []SlotDescriptor{{ID: id1}, {ID: id2}}); err != nil {
					t.Fatalf("setup InstantiateSlots() error = %v", err)
				}
				return wsID, []plugin.ID{id1, id2}
			},
			ctx:         context.Background,
			expectedErr: nil,
		},
		{
			name: "missing workspace returns NotFound",
			setup: func(_ *testing.T, _ *Plugin, _ *plugin.Registry, _ *workspace.Registry) (workspace.ID, []plugin.ID) {
				return 999, nil
			},
			ctx:         context.Background,
			expectedErr: workspace.ErrNotFound,
		},
		{
			name: "canceled context returns error",
			setup: func(t *testing.T, _ *Plugin, _ *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []plugin.ID) {
				return mustCreateWorkspace(t, wsRepo, "alpha"), nil
			},
			ctx:         canceledContext,
			expectedErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plgRepo := plugin.NewRegistry()
			wsRepo := workspace.NewRegistry()
			svc := NewPlugin(plgRepo, wsRepo)

			wsID, wantIDs := tt.setup(t, svc, plgRepo, wsRepo)
			got, err := svc.ListSlots(tt.ctx(), wsID)
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("error = %v, want %v", err, tt.expectedErr)
			}
			if tt.expectedErr == nil {
				slices.Sort(got)
				slices.Sort(wantIDs)
				if !slices.Equal(got, wantIDs) {
					t.Errorf("slots = %v, want %v", got, wantIDs)
				}
			}
		})
	}
}

func TestPluginService_DestroySlots(t *testing.T) {
	errClose := errors.New("close failed")

	tests := []struct {
		name        string
		setup       func(t *testing.T, svc *Plugin, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []plugin.ID)
		ctx         func() context.Context
		expectedErr error
		verify      func(t *testing.T, svc *Plugin, wsID workspace.ID)
	}{
		{
			name: "destroys specified slots and closes tracked closers",
			setup: func(t *testing.T, svc *Plugin, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []plugin.ID) {
				wsID := mustCreateWorkspace(t, wsRepo, "alpha")
				closer1 := &stubCloser{}
				closer2 := &stubCloser{}
				id1 := mustRegisterPlugin(t, plgRepo, "p1", &stubPlugin{
					initFn: func(_ context.Context, _ plugin.InitConfig, slot *plugin.Slot) error {
						slot.TrackCloser(closer1)
						return nil
					},
				})
				id2 := mustRegisterPlugin(t, plgRepo, "p2", &stubPlugin{
					initFn: func(_ context.Context, _ plugin.InitConfig, slot *plugin.Slot) error {
						slot.TrackCloser(closer2)
						return nil
					},
				})
				if err := svc.InstantiateSlots(context.Background(), wsID, []SlotDescriptor{{ID: id1}, {ID: id2}}); err != nil {
					t.Fatalf("setup InstantiateSlots() error = %v", err)
				}
				t.Cleanup(func() {
					if closer1.closed.Load() != 1 {
						t.Errorf("closer1.closed = %d, want 1", closer1.closed.Load())
					}
					if closer2.closed.Load() != 0 {
						t.Errorf("closer2.closed = %d, want 0", closer2.closed.Load())
					}
				})
				return wsID, []plugin.ID{id1}
			},
			ctx:         context.Background,
			expectedErr: nil,
			verify: func(t *testing.T, svc *Plugin, wsID workspace.ID) {
				got, err := svc.ListSlots(context.Background(), wsID)
				if err != nil {
					t.Fatalf("ListSlots() error = %v", err)
				}
				if len(got) != 1 {
					t.Errorf("len(ListSlots()) = %d, want 1", len(got))
				}
			},
		},
		{
			name: "destroying non-existent slot is a no-op",
			setup: func(t *testing.T, _ *Plugin, _ *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []plugin.ID) {
				return mustCreateWorkspace(t, wsRepo, "alpha"), []plugin.ID{999}
			},
			ctx:         context.Background,
			expectedErr: nil,
		},
		{
			name: "closer error is returned",
			setup: func(t *testing.T, svc *Plugin, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []plugin.ID) {
				wsID := mustCreateWorkspace(t, wsRepo, "alpha")
				id1 := mustRegisterPlugin(t, plgRepo, "p1", &stubPlugin{
					initFn: func(_ context.Context, _ plugin.InitConfig, slot *plugin.Slot) error {
						slot.TrackCloser(&stubCloser{err: errClose})
						return nil
					},
				})
				if err := svc.InstantiateSlots(context.Background(), wsID, []SlotDescriptor{{ID: id1}}); err != nil {
					t.Fatalf("setup InstantiateSlots() error = %v", err)
				}
				return wsID, []plugin.ID{id1}
			},
			ctx:         context.Background,
			expectedErr: errClose,
		},
		{
			name: "missing workspace returns NotFound",
			setup: func(_ *testing.T, _ *Plugin, _ *plugin.Registry, _ *workspace.Registry) (workspace.ID, []plugin.ID) {
				return 999, []plugin.ID{1}
			},
			ctx:         context.Background,
			expectedErr: workspace.ErrNotFound,
		},
		{
			name: "canceled context returns error",
			setup: func(t *testing.T, _ *Plugin, _ *plugin.Registry, wsRepo *workspace.Registry) (workspace.ID, []plugin.ID) {
				return mustCreateWorkspace(t, wsRepo, "alpha"), []plugin.ID{1}
			},
			ctx:         canceledContext,
			expectedErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plgRepo := plugin.NewRegistry()
			wsRepo := workspace.NewRegistry()
			svc := NewPlugin(plgRepo, wsRepo)

			wsID, ids := tt.setup(t, svc, plgRepo, wsRepo)
			err := svc.DestroySlots(tt.ctx(), wsID, ids)
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("error = %v, want %v", err, tt.expectedErr)
			}
			if tt.verify != nil {
				tt.verify(t, svc, wsID)
			}
		})
	}
}

func TestPluginService_ListPlugins(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, plgRepo *plugin.Registry) map[plugin.Name]plugin.ID
		ctx         func() context.Context
		expectedErr error
	}{
		{
			name: "empty registry returns no plugins",
			setup: func(_ *testing.T, _ *plugin.Registry) map[plugin.Name]plugin.ID {
				return map[plugin.Name]plugin.ID{}
			},
			ctx:         context.Background,
			expectedErr: nil,
		},
		{
			name: "returns registered plugin IDs and name mapping",
			setup: func(t *testing.T, plgRepo *plugin.Registry) map[plugin.Name]plugin.ID {
				id1 := mustRegisterPlugin(t, plgRepo, "alpha", &stubPlugin{})
				id2 := mustRegisterPlugin(t, plgRepo, "beta", &stubPlugin{})
				return map[plugin.Name]plugin.ID{
					"alpha": id1,
					"beta":  id2,
				}
			},
			ctx:         context.Background,
			expectedErr: nil,
		},
		{
			name: "canceled context returns error",
			setup: func(_ *testing.T, _ *plugin.Registry) map[plugin.Name]plugin.ID {
				return nil
			},
			ctx:         canceledContext,
			expectedErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plgRepo := plugin.NewRegistry()
			wsRepo := workspace.NewRegistry()
			svc := NewPlugin(plgRepo, wsRepo)

			wantNames := tt.setup(t, plgRepo)
			ids, nameMap, err := svc.ListPlugins(tt.ctx())
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("error = %v, want %v", err, tt.expectedErr)
			}
			if tt.expectedErr == nil {
				if len(ids) != len(wantNames) {
					t.Errorf("len(ids) = %d, want %d", len(ids), len(wantNames))
				}
				for name, wantID := range wantNames {
					if gotID, ok := nameMap[name]; !ok || gotID != wantID {
						t.Errorf("nameMap[%q] = %d (ok=%v), want %d", name, gotID, ok, wantID)
					}
					if !slices.Contains(ids, wantID) {
						t.Errorf("ids %v missing %d", ids, wantID)
					}
				}
			}
		})
	}
}

type stubPlugin struct {
	initFn func(ctx context.Context, config plugin.InitConfig, slot *plugin.Slot) error
}

func (s *stubPlugin) Init(ctx context.Context, config plugin.InitConfig, slot *plugin.Slot) error {
	if s.initFn != nil {
		return s.initFn(ctx, config, slot)
	}
	return nil
}

type stubCloser struct {
	closed atomic.Int32
	err    error
}

func (s *stubCloser) Close() error {
	s.closed.Add(1)
	return s.err
}

func mustCreateWorkspace(t *testing.T, wsRepo *workspace.Registry, name workspace.Name) workspace.ID {
	t.Helper()
	id, err := wsRepo.Put(name, workspace.NewWorkspace())
	if err != nil {
		t.Fatalf("failed to create workspace %q: %v", name, err)
	}
	return id
}

func mustRegisterPlugin(t *testing.T, plgRepo *plugin.Registry, name plugin.Name, plg plugin.Plugin) plugin.ID {
	t.Helper()
	id, err := plgRepo.Put(name, plg)
	if err != nil {
		t.Fatalf("failed to register plugin %q: %v", name, err)
	}
	return id
}
