package service

import (
	"context"
	"fmt"

	"github.com/Skoowshot/qrix-runtime/internal/env"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
)

type Environment struct {
	sessionRepo *env.SessionRegistry
}

func (e *Environment) Connect(ctx context.Context, wsID workspace.ID, name env.Name, metadata env.Metadata) (env.SessionToken, error) {
	token := env.GenerateSessionToken()
	_, err := e.sessionRepo.PutPending(ctx, name, token, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to put session: %w", err)
	}
	return "", fmt.Errorf("not implemented")
}
