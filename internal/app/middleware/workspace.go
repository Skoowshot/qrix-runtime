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
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}
		vals := md.Get("workspace")
		if len(vals) == 0 {
			return handler(ctx, req)
		}
		wsID, err := strconv.ParseUint(vals[0], 10, 64)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid workspace")
		}
		ctx = context.WithValue(ctx, wsIDKey{}, workspace.ID(wsID))
		return handler(ctx, req)
	}
}

func WorkspaceFromContext(ctx context.Context) (workspace.ID, bool) {
	id, ok := ctx.Value(wsIDKey{}).(workspace.ID)
	return id, ok
}
