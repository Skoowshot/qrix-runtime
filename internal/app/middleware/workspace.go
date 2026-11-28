package middleware

import (
	"context"
	"strconv"

	"github.com/Skoowshot/qrix-runtime/internal/workspace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type wsIDKey struct{}

func Workspace() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, err := withWorkspace(ctx)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func StreamWorkspace() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, err := withWorkspace(ss.Context())
		if err != nil {
			return err
		}
		return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
	}
}

func withWorkspace(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, nil
	}
	vals := md.Get("workspace")
	if len(vals) == 0 {
		return ctx, nil
	}
	wsID, err := strconv.ParseUint(vals[0], 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid workspace")
	}
	return context.WithValue(ctx, wsIDKey{}, workspace.ID(wsID)), nil
}

func WorkspaceFromContext(ctx context.Context) (workspace.ID, bool) {
	id, ok := ctx.Value(wsIDKey{}).(workspace.ID)
	return id, ok
}
