package workspace

import (
	"context"
	"fmt"

	"github.com/Skoowshot/qrix-runtime/internal/store"
)

type Service struct {
	registry *Registry
}

func NewService(registry *Registry) *Service {
	return &Service{
		registry: registry,
	}
}

func (s *Service) Create(ctx context.Context, name Name) (ID, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	ws := &Workspace{
		Stores:  store.NewRegistry(),
		Network: nil,
	}

	id, err := s.registry.Put(name, ws)
	if err != nil {
		return 0, fmt.Errorf("put workspace to registry: %w", err)
	}
	return id, nil
}

func (s *Service) GetByName(ctx context.Context, name Name) (ID, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	id, ok := s.registry.GetID(name)
	if !ok {
		return 0, ErrNotFound
	}
	return id, nil
}
