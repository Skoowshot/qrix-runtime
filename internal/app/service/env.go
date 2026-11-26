package service

import (
	"context"
	"fmt"

	"github.com/Skoowshot/qrix-runtime/internal/env"
	"github.com/Skoowshot/qrix-runtime/internal/rpc"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
)

type Environment struct {
	wsRepo *workspace.Registry
}

func NewEnvironment(workspaceRepo *workspace.Registry) *Environment {
	return &Environment{
		wsRepo: workspaceRepo,
	}
}

func (e *Environment) Connect(ctx context.Context, wsID workspace.ID, name env.Name, metadata env.Metadata) (env.SessionToken, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	ws, ok := e.wsRepo.Get(wsID)
	if !ok {
		return "", fmt.Errorf("%w: %d", workspace.ErrNotFound, wsID)
	}

	token := env.GenerateSessionToken()
	if _, err := ws.Sessions.PutPending(name, token, metadata); err != nil {
		return "", err
	}

	return token, nil
}

func (e *Environment) OpenCall(ctx context.Context, wsID workspace.ID, token env.SessionToken) (rpc.CallPipe, error) {
	if err := ctx.Err(); err != nil {
		return rpc.CallPipe{}, err
	}

	ws, ok := e.wsRepo.Get(wsID)
	if !ok {
		return rpc.CallPipe{}, fmt.Errorf("%w: %d", workspace.ErrNotFound, wsID)
	}

	in := make(chan rpc.CallFrame)
	out := make(chan rpc.CallFrame)
	if err := ws.Sessions.AttachCall(token, in, out); err != nil {
		return rpc.CallPipe{}, err
	}
	pipe := rpc.CallPipe{
		In:  in,
		Out: out,
	}
	return pipe, nil
}
