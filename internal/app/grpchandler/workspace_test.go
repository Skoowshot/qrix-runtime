package grpchandler

import (
	"context"
	"net"
	"testing"

	workspacev1 "github.com/Skoowshot/qrix-runtime/gen/go/workspace/v1"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestWorkspace_Create(t *testing.T) {
	tests := []struct {
		name         string
		req          *workspacev1.CreateRequest
		setup        func(t *testing.T, client workspacev1.WorkspaceServiceClient)
		expectedCode codes.Code
	}{
		{
			name:         "created successfully",
			req:          &workspacev1.CreateRequest{Name: "alpha"},
			setup:        nil,
			expectedCode: codes.OK,
		},
		{
			name:         "duplicate name returns AlreadyExists",
			req:          &workspacev1.CreateRequest{Name: "alpha"},
			setup:        setupGRPCWorkspace("alpha"),
			expectedCode: codes.AlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := setupClient(t)
			if tt.setup != nil {
				tt.setup(t, client)
			}

			resp, err := client.Create(context.Background(), tt.req)
			if status.Code(err) != tt.expectedCode {
				t.Errorf("code = %v, want %v (err: %v)", status.Code(err), tt.expectedCode, err)
			}
			if tt.expectedCode == codes.OK && resp.GetId() == 0 {
				t.Errorf("id = 0, want non-zero")
			}
		})
	}
}

func TestWorkspace_GetByName(t *testing.T) {
	tests := []struct {
		name         string
		req          *workspacev1.GetByNameRequest
		setup        func(t *testing.T, client workspacev1.WorkspaceServiceClient)
		expectedCode codes.Code
	}{
		{
			name:         "existing workspace found",
			req:          &workspacev1.GetByNameRequest{Name: "alpha"},
			setup:        setupGRPCWorkspace("alpha"),
			expectedCode: codes.OK,
		},
		{
			name:         "missing workspace returns NotFound",
			req:          &workspacev1.GetByNameRequest{Name: "missing"},
			setup:        nil,
			expectedCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := setupClient(t)
			if tt.setup != nil {
				tt.setup(t, client)
			}

			resp, err := client.GetByName(context.Background(), tt.req)
			if status.Code(err) != tt.expectedCode {
				t.Errorf("code = %v, want %v (err: %v)", status.Code(err), tt.expectedCode, err)
			}
			if tt.expectedCode == codes.OK && resp.GetId() == 0 {
				t.Errorf("id = 0, want non-zero")
			}
		})
	}
}

func setupClient(t *testing.T) workspacev1.WorkspaceServiceClient {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()

	wsSvc := workspace.NewService(workspace.NewRegistry())
	workspacev1.RegisterWorkspaceServiceServer(srv, NewWorkspace(wsSvc))

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

	return workspacev1.NewWorkspaceServiceClient(conn)
}

func setupGRPCWorkspace(name string) func(t *testing.T, client workspacev1.WorkspaceServiceClient) {
	return func(t *testing.T, client workspacev1.WorkspaceServiceClient) {
		t.Helper()
		if _, err := client.Create(context.Background(), &workspacev1.CreateRequest{Name: name}); err != nil {
			t.Fatalf("failed to setup workspace %q: %v", name, err)
		}
	}
}
