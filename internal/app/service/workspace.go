package service

import (
	"context"
	"fmt"

	"github.com/Skoowshot/qrix-runtime/internal/workspace"
)

type Workspace struct {
	registry *workspace.Registry
}

func NewWorkspace(registry *workspace.Registry) *Workspace {
	return &Workspace{
		registry: registry,
	}
}

func (s *Workspace) Create(ctx context.Context, name workspace.Name) (workspace.ID, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	ws := workspace.NewWorkspace()

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
