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
			return nil, status.Errorf(codes.InvalidArgument, "missing metadata")
		}
		vals := md.Get("workspace")
		if len(vals) == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "missing workspace")
		}
		wsID, err := strconv.ParseUint(vals[0], 10, 64)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid workspace")
		}
		ctx = context.WithValue(ctx, wsIDKey{}, workspace.ID(wsID))
		return handler(ctx, req)
	}
}

func WorkspaceFromContext(ctx context.Context) (workspace.ID, bool) {
	id, ok := ctx.Value(wsIDKey{}).(workspace.ID)
	return id, ok
}
