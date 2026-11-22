package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Skoowshot/qrix-runtime/internal/workspace"
)

type Workspace struct {
	sessionTTL time.Duration
	registry   *workspace.Registry
}

func NewWorkspace(sessionTTL time.Duration, registry *workspace.Registry) *Workspace {
	return &Workspace{
		sessionTTL: sessionTTL,
		registry:   registry,
	}
}

func (s *Workspace) Create(ctx context.Context, name workspace.Name) (workspace.ID, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	ws := workspace.NewWorkspace(s.sessionTTL)

	id, err := s.registry.Put(name, ws)
	if err != nil {
		return 0, fmt.Errorf("put workspace to registry: %w", err)
	}
	return id, nil
}

func (s *Workspace) GetByName(ctx context.Context, name workspace.Name) (workspace.ID, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	id, ok := s.registry.GetID(name)
	if !ok {
		return 0, workspace.ErrNotFound
	}
	return id, nil
}
