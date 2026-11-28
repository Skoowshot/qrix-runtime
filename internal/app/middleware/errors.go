package middleware

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Skoowshot/qrix-runtime/internal/env"
	"github.com/Skoowshot/qrix-runtime/internal/plugin"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Error() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}
		return nil, mapError(ctx, info.FullMethod, err)
	}
}

func StreamError() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, ss)
		if err == nil {
			return nil
		}
		return mapError(ss.Context(), info.FullMethod, err)
	}
}

func mapError(ctx context.Context, method string, err error) error {
	if _, ok := status.FromError(err); ok {
		return err
	}

	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, err.Error())
	case errors.Is(err, workspace.ErrNotFound),
		errors.Is(err, plugin.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, plugin.ErrSlotAlreadyExists),
		errors.Is(err, workspace.ErrNameAlreadyExists),
		errors.Is(err, plugin.ErrNameAlreadyExists),
		errors.Is(err, env.ErrNameAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, env.ErrSessionInvalid):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, env.ErrSessionNotPending):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		slog.ErrorContext(ctx, "internal server error", "method", method, "err", err)
		return status.Error(codes.Internal, "internal server error")
	}
}
