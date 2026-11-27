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

// Error returns a unary server interceptor that maps context and domain errors
// to gRPC status errors, falling back to codes.Internal for unknown errors.
func Error() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}

		if _, ok := status.FromError(err); ok {
			return nil, err
		}

		switch {
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, err.Error())
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, err.Error())
		case errors.Is(err, workspace.ErrNotFound),
			errors.Is(err, plugin.ErrNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, plugin.ErrSlotAlreadyExists),
			errors.Is(err, workspace.ErrNameAlreadyExists),
			errors.Is(err, plugin.ErrNameAlreadyExists):
			return nil, status.Error(codes.AlreadyExists, err.Error())
		case errors.Is(err, env.ErrSessionInvalid):
			return nil, status.Error(codes.Unauthenticated, err.Error())
		case errors.Is(err, env.ErrSessionNotPending):
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		default:
			slog.ErrorContext(ctx, "internal server error", "method", info.FullMethod, "err", err)
			return nil, status.Error(codes.Internal, "internal server error")
		}
	}
}
