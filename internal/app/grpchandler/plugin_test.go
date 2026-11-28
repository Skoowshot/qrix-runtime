package grpchandler

import (
	"context"
	"errors"
	"io"
	"net"
	"slices"
	"strconv"
	"testing"
	"time"

	pluginv1 "github.com/Skoowshot/qrix-runtime/gen/go/plugin/v1"
	"github.com/Skoowshot/qrix-runtime/internal/app/middleware"
	"github.com/Skoowshot/qrix-runtime/internal/app/service"
	"github.com/Skoowshot/qrix-runtime/internal/plugin"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestPlugin_InstantiateSlots(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, client pluginv1.PluginServiceClient, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (context.Context, *pluginv1.InstantiateSlotsRequest)
		expectedCode codes.Code
	}{
		{
			name: "instantiated successfully with valid and empty config",
			setup: func(t *testing.T, _ pluginv1.PluginServiceClient, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (context.Context, *pluginv1.InstantiateSlotsRequest) {
				wsID := mustCreateGRPCWorkspace(t, wsRepo, "alpha")
				var gotCfg plugin.Config
				id1 := mustRegisterGRPCPlugin(t, plgRepo, "p1", &grpcStubPlugin{
					initFn: func(_ context.Context, cfg plugin.InitConfig, _ *plugin.Slot) error {
						gotCfg = cfg.Config
						return nil
					},
				})
				id2 := mustRegisterGRPCPlugin(t, plgRepo, "p2", &grpcStubPlugin{})
				t.Cleanup(func() {
					if string(gotCfg) != "{}" {
						t.Errorf("empty config = %q, want {}", string(gotCfg))
					}
				})
				return ctxWithWorkspace(wsID), &pluginv1.InstantiateSlotsRequest{
					Descriptors: []*pluginv1.SlotDescriptor{
						{PluginId: uint64(id1)},
						{PluginId: uint64(id2), Config: []byte(`{"key":"val"}`)},
					},
				}
			},
			expectedCode: codes.OK,
		},
		{
			name: "missing workspace in context returns InvalidArgument",
			setup: func(_ *testing.T, _ pluginv1.PluginServiceClient, _ *plugin.Registry, _ *workspace.Registry) (context.Context, *pluginv1.InstantiateSlotsRequest) {
				return context.Background(), &pluginv1.InstantiateSlotsRequest{}
			},
			expectedCode: codes.InvalidArgument,
		},
		{
			name: "invalid config JSON returns InvalidArgument",
			setup: func(t *testing.T, _ pluginv1.PluginServiceClient, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (context.Context, *pluginv1.InstantiateSlotsRequest) {
				wsID := mustCreateGRPCWorkspace(t, wsRepo, "alpha")
				id1 := mustRegisterGRPCPlugin(t, plgRepo, "p1", &grpcStubPlugin{})
				return ctxWithWorkspace(wsID), &pluginv1.InstantiateSlotsRequest{
					Descriptors: []*pluginv1.SlotDescriptor{
						{PluginId: uint64(id1), Config: []byte("{invalid")},
					},
				}
			},
			expectedCode: codes.InvalidArgument,
		},
		{
			name: "missing workspace in registry returns NotFound",
			setup: func(t *testing.T, _ pluginv1.PluginServiceClient, plgRepo *plugin.Registry, _ *workspace.Registry) (context.Context, *pluginv1.InstantiateSlotsRequest) {
				id1 := mustRegisterGRPCPlugin(t, plgRepo, "p1", &grpcStubPlugin{})
				return ctxWithWorkspace(999), &pluginv1.InstantiateSlotsRequest{
					Descriptors: []*pluginv1.SlotDescriptor{{PluginId: uint64(id1)}},
				}
			},
			expectedCode: codes.NotFound,
		},
		{
			name: "missing plugin returns NotFound",
			setup: func(t *testing.T, _ pluginv1.PluginServiceClient, _ *plugin.Registry, wsRepo *workspace.Registry) (context.Context, *pluginv1.InstantiateSlotsRequest) {
				wsID := mustCreateGRPCWorkspace(t, wsRepo, "alpha")
				return ctxWithWorkspace(wsID), &pluginv1.InstantiateSlotsRequest{
					Descriptors: []*pluginv1.SlotDescriptor{{PluginId: 999}},
				}
			},
			expectedCode: codes.NotFound,
		},
		{
			name: "duplicate slot returns AlreadyExists",
			setup: func(t *testing.T, client pluginv1.PluginServiceClient, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (context.Context, *pluginv1.InstantiateSlotsRequest) {
				wsID := mustCreateGRPCWorkspace(t, wsRepo, "alpha")
				id1 := mustRegisterGRPCPlugin(t, plgRepo, "p1", &grpcStubPlugin{})
				ctx := ctxWithWorkspace(wsID)
				req := &pluginv1.InstantiateSlotsRequest{
					Descriptors: []*pluginv1.SlotDescriptor{{PluginId: uint64(id1)}},
				}
				if _, err := client.InstantiateSlots(ctx, req); err != nil {
					t.Fatalf("setup InstantiateSlots() error = %v", err)
				}
				return ctx, req
			},
			expectedCode: codes.AlreadyExists,
		},
		{
			name: "plugin init failure returns Internal",
			setup: func(t *testing.T, _ pluginv1.PluginServiceClient, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (context.Context, *pluginv1.InstantiateSlotsRequest) {
				wsID := mustCreateGRPCWorkspace(t, wsRepo, "alpha")
				id1 := mustRegisterGRPCPlugin(t, plgRepo, "p1", &grpcStubPlugin{
					initFn: func(_ context.Context, _ plugin.InitConfig, _ *plugin.Slot) error {
						return errors.New("boom")
					},
				})
				return ctxWithWorkspace(wsID), &pluginv1.InstantiateSlotsRequest{
					Descriptors: []*pluginv1.SlotDescriptor{{PluginId: uint64(id1)}},
				}
			},
			expectedCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, plgRepo, wsRepo := setupPluginClient(t)
			ctx, req := tt.setup(t, client, plgRepo, wsRepo)

			_, err := client.InstantiateSlots(ctx, req)
			if status.Code(err) != tt.expectedCode {
				t.Errorf("code = %v, want %v (err: %v)", status.Code(err), tt.expectedCode, err)
			}
		})
	}
}

func TestPlugin_ListSlots(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, client pluginv1.PluginServiceClient, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (context.Context, []uint64)
		expectedCode codes.Code
	}{
		{
			name: "lists instantiated slots",
			setup: func(t *testing.T, client pluginv1.PluginServiceClient, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (context.Context, []uint64) {
				wsID := mustCreateGRPCWorkspace(t, wsRepo, "alpha")
				id1 := mustRegisterGRPCPlugin(t, plgRepo, "p1", &grpcStubPlugin{})
				id2 := mustRegisterGRPCPlugin(t, plgRepo, "p2", &grpcStubPlugin{})
				ctx := ctxWithWorkspace(wsID)
				if _, err := client.InstantiateSlots(ctx, &pluginv1.InstantiateSlotsRequest{
					Descriptors: []*pluginv1.SlotDescriptor{
						{PluginId: uint64(id1)},
						{PluginId: uint64(id2)},
					},
				}); err != nil {
					t.Fatalf("setup InstantiateSlots() error = %v", err)
				}
				return ctx, []uint64{uint64(id1), uint64(id2)}
			},
			expectedCode: codes.OK,
		},
		{
			name: "missing workspace in context returns InvalidArgument",
			setup: func(_ *testing.T, _ pluginv1.PluginServiceClient, _ *plugin.Registry, _ *workspace.Registry) (context.Context, []uint64) {
				return context.Background(), nil
			},
			expectedCode: codes.InvalidArgument,
		},
		{
			name: "missing workspace in registry returns NotFound",
			setup: func(_ *testing.T, _ pluginv1.PluginServiceClient, _ *plugin.Registry, _ *workspace.Registry) (context.Context, []uint64) {
				return ctxWithWorkspace(999), nil
			},
			expectedCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, plgRepo, wsRepo := setupPluginClient(t)
			ctx, wantIDs := tt.setup(t, client, plgRepo, wsRepo)

			resp, err := client.ListSlots(ctx, &pluginv1.ListSlotsRequest{})
			if status.Code(err) != tt.expectedCode {
				t.Errorf("code = %v, want %v (err: %v)", status.Code(err), tt.expectedCode, err)
			}
			if tt.expectedCode == codes.OK {
				got := resp.GetPluginIds()
				slices.Sort(got)
				slices.Sort(wantIDs)
				if !slices.Equal(got, wantIDs) {
					t.Errorf("PluginIds = %v, want %v", got, wantIDs)
				}
			}
		})
	}
}

func TestPlugin_DestroySlots(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, client pluginv1.PluginServiceClient, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (context.Context, *pluginv1.DestroySlotsRequest)
		expectedCode codes.Code
	}{
		{
			name: "destroys specified slots",
			setup: func(t *testing.T, client pluginv1.PluginServiceClient, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (context.Context, *pluginv1.DestroySlotsRequest) {
				wsID := mustCreateGRPCWorkspace(t, wsRepo, "alpha")
				id1 := mustRegisterGRPCPlugin(t, plgRepo, "p1", &grpcStubPlugin{})
				ctx := ctxWithWorkspace(wsID)
				if _, err := client.InstantiateSlots(ctx, &pluginv1.InstantiateSlotsRequest{
					Descriptors: []*pluginv1.SlotDescriptor{{PluginId: uint64(id1)}},
				}); err != nil {
					t.Fatalf("setup InstantiateSlots() error = %v", err)
				}
				return ctx, &pluginv1.DestroySlotsRequest{PluginIds: []uint64{uint64(id1)}}
			},
			expectedCode: codes.OK,
		},
		{
			name: "missing workspace in context returns InvalidArgument",
			setup: func(_ *testing.T, _ pluginv1.PluginServiceClient, _ *plugin.Registry, _ *workspace.Registry) (context.Context, *pluginv1.DestroySlotsRequest) {
				return context.Background(), &pluginv1.DestroySlotsRequest{PluginIds: []uint64{1}}
			},
			expectedCode: codes.InvalidArgument,
		},
		{
			name: "missing workspace in registry returns NotFound",
			setup: func(_ *testing.T, _ pluginv1.PluginServiceClient, _ *plugin.Registry, _ *workspace.Registry) (context.Context, *pluginv1.DestroySlotsRequest) {
				return ctxWithWorkspace(999), &pluginv1.DestroySlotsRequest{PluginIds: []uint64{1}}
			},
			expectedCode: codes.NotFound,
		},
		{
			name: "closer failure returns Internal",
			setup: func(t *testing.T, client pluginv1.PluginServiceClient, plgRepo *plugin.Registry, wsRepo *workspace.Registry) (context.Context, *pluginv1.DestroySlotsRequest) {
				wsID := mustCreateGRPCWorkspace(t, wsRepo, "alpha")
				id1 := mustRegisterGRPCPlugin(t, plgRepo, "p1", &grpcStubPlugin{
					initFn: func(_ context.Context, _ plugin.InitConfig, slot *plugin.Slot) error {
						slot.TrackCloser(errCloser{})
						return nil
					},
				})
				ctx := ctxWithWorkspace(wsID)
				if _, err := client.InstantiateSlots(ctx, &pluginv1.InstantiateSlotsRequest{
					Descriptors: []*pluginv1.SlotDescriptor{{PluginId: uint64(id1)}},
				}); err != nil {
					t.Fatalf("setup InstantiateSlots() error = %v", err)
				}
				return ctx, &pluginv1.DestroySlotsRequest{PluginIds: []uint64{uint64(id1)}}
			},
			expectedCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, plgRepo, wsRepo := setupPluginClient(t)
			ctx, req := tt.setup(t, client, plgRepo, wsRepo)

			_, err := client.DestroySlots(ctx, req)
			if status.Code(err) != tt.expectedCode {
				t.Errorf("code = %v, want %v (err: %v)", status.Code(err), tt.expectedCode, err)
			}
		})
	}
}

func TestPlugin_ListPlugins(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T, plgRepo *plugin.Registry) map[string]uint64
		expectedCode codes.Code
	}{
		{
			name: "empty registry returns empty response",
			setup: func(_ *testing.T, _ *plugin.Registry) map[string]uint64 {
				return map[string]uint64{}
			},
			expectedCode: codes.OK,
		},
		{
			name: "returns registered plugins and name mapping",
			setup: func(t *testing.T, plgRepo *plugin.Registry) map[string]uint64 {
				id1 := mustRegisterGRPCPlugin(t, plgRepo, "alpha", &grpcStubPlugin{})
				id2 := mustRegisterGRPCPlugin(t, plgRepo, "beta", &grpcStubPlugin{})
				return map[string]uint64{
					"alpha": uint64(id1),
					"beta":  uint64(id2),
				}
			},
			expectedCode: codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, plgRepo, _ := setupPluginClient(t)
			wantNames := tt.setup(t, plgRepo)

			resp, err := client.ListPlugins(context.Background(), &pluginv1.ListPluginsRequest{})
			if status.Code(err) != tt.expectedCode {
				t.Errorf("code = %v, want %v (err: %v)", status.Code(err), tt.expectedCode, err)
			}
			if tt.expectedCode == codes.OK {
				if len(resp.GetIds()) != len(wantNames) {
					t.Errorf("len(Ids) = %d, want %d", len(resp.GetIds()), len(wantNames))
				}
				for name, wantID := range wantNames {
					if gotID, ok := resp.GetNameToId()[name]; !ok || gotID != wantID {
						t.Errorf("NameToId[%q] = %d (ok=%v), want %d", name, gotID, ok, wantID)
					}
					if !slices.Contains(resp.GetIds(), wantID) {
						t.Errorf("Ids %v missing %d", resp.GetIds(), wantID)
					}
				}
			}
		})
	}
}

func setupPluginClient(t *testing.T) (pluginv1.PluginServiceClient, *plugin.Registry, *workspace.Registry) {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(
		middleware.Error(),
		middleware.Workspace(),
	))

	plgRepo := plugin.NewRegistry()
	wsRepo := workspace.NewRegistry()
	plgSvc := service.NewPlugin(plgRepo, wsRepo)
	pluginv1.RegisterPluginServiceServer(srv, NewPlugin(plgSvc))

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return pluginv1.NewPluginServiceClient(conn), plgRepo, wsRepo
}

func ctxWithWorkspace(wsID workspace.ID) context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), "workspace", strconv.FormatUint(uint64(wsID), 10))
}

func mustCreateGRPCWorkspace(t *testing.T, wsRepo *workspace.Registry, name workspace.Name) workspace.ID {
	t.Helper()
	id, err := wsRepo.Put(name, workspace.NewWorkspace(15*time.Second))
	if err != nil {
		t.Fatalf("failed to create workspace %q: %v", name, err)
	}
	return id
}

func mustRegisterGRPCPlugin(t *testing.T, plgRepo *plugin.Registry, name plugin.Name, plg plugin.Plugin) plugin.ID {
	t.Helper()
	id, err := plgRepo.Put(name, plg)
	if err != nil {
		t.Fatalf("failed to register plugin %q: %v", name, err)
	}
	return id
}

type grpcStubPlugin struct {
	initFn func(ctx context.Context, config plugin.InitConfig, slot *plugin.Slot) error
}

func (s *grpcStubPlugin) Init(ctx context.Context, config plugin.InitConfig, slot *plugin.Slot) error {
	if s.initFn != nil {
		return s.initFn(ctx, config, slot)
	}
	return nil
}

type errCloser struct{}

func (errCloser) Close() error {
	return io.ErrUnexpectedEOF
}
