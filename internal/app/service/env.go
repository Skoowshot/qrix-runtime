package service

import (
	"context"
	"fmt"

	"github.com/Skoowshot/qrix-runtime/internal/env"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
)

type Environment struct {
	workspaceRepo *workspace.Registry
}

func (e *Environment) Connect(ctx context.Context, wsID workspace.ID, name env.Name, metadata env.Metadata) (env.SessionToken, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	ws, ok := e.workspaceRepo.Get(wsID)
	if !ok {
		return "", fmt.Errorf("%w: %d", workspace.ErrNotFound, wsID)
	}

	token := env.GenerateSessionToken()
	if _, err := ws.Sessions.PutPending(name, token, metadata); err != nil {
		return "", err
	}

	return token, nil
}
