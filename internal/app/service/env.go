package service

import (
	"context"
	"fmt"

	"github.com/Skoowshot/qrix-runtime/internal/env"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
)

type Environment struct{}

func (e *Environment) Connect(ctx context.Context, wsID workspace.ID, name env.Name, metadata env.Metadata) (env.SessionToken, error) {
	return "", fmt.Errorf("not implemented")
}
