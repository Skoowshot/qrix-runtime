package workspace

import (
	"context"
	"errors"
	"testing"
)

func TestService_Create(t *testing.T) {
	tests := []struct {
		name        string
		wsName      Name
		setup       func(t *testing.T, svc *Service)
		ctx         func() context.Context
		expectedErr error
	}{
		{
			name:        "new workspace created successfully",
			wsName:      "alpha",
			setup:       nil,
			ctx:         context.Background,
			expectedErr: nil,
		},
		{
			name:        "duplicate workspace name returns error",
			wsName:      "alpha",
			setup:       setupWorkspace("alpha"),
			ctx:         context.Background,
			expectedErr: ErrNameAlreadyExists,
		},
		{
			name:        "canceled context returns error",
			wsName:      "alpha",
			setup:       nil,
			ctx:         canceledContext,
			expectedErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(NewRegistry())
			if tt.setup != nil {
				tt.setup(t, svc)
			}

			id, err := svc.Create(tt.ctx(), tt.wsName)
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("error = %v, want %v", err, tt.expectedErr)
			}
			if tt.expectedErr == nil && id == 0 {
				t.Errorf("id = 0, want non-zero")
			}
		})
	}
}

func TestService_GetByName(t *testing.T) {
	tests := []struct {
		name        string
		wsName      Name
		setup       func(t *testing.T, svc *Service)
		ctx         func() context.Context
		expectedErr error
	}{
		{
			name:        "existing workspace found",
			wsName:      "alpha",
			setup:       setupWorkspace("alpha"),
			ctx:         context.Background,
			expectedErr: nil,
		},
		{
			name:        "missing workspace returns not found",
			wsName:      "missing",
			setup:       nil,
			ctx:         context.Background,
			expectedErr: ErrNotFound,
		},
		{
			name:        "canceled context returns error",
			wsName:      "alpha",
			setup:       setupWorkspace("alpha"),
			ctx:         canceledContext,
			expectedErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(NewRegistry())
			if tt.setup != nil {
				tt.setup(t, svc)
			}

			id, err := svc.GetByName(tt.ctx(), tt.wsName)
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("error = %v, want %v", err, tt.expectedErr)
			}
			if tt.expectedErr == nil && id == 0 {
				t.Errorf("id = 0, want non-zero")
			}
		})
	}
}

func setupWorkspace(name Name) func(t *testing.T, svc *Service) {
	return func(t *testing.T, svc *Service) {
		t.Helper()
		if _, err := svc.Create(context.Background(), name); err != nil {
			t.Fatalf("failed to setup workspace %q: %v", name, err)
		}
	}
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
