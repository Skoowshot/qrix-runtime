package middleware

import (
	"context"

	"github.com/Skoowshot/qrix-runtime/internal/env"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type sessTokenKey struct{}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

func StreamSessionToken() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(srv, ss)
		}
		vals := md.Get("session-token")
		if len(vals) == 0 || vals[0] == "" {
			return handler(srv, ss)
		}
		ctx = context.WithValue(ctx, sessTokenKey{}, env.SessionToken(vals[0]))
		return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
	}
}

func SessionTokenFromContext(ctx context.Context) (env.SessionToken, bool) {
	token, ok := ctx.Value(sessTokenKey{}).(env.SessionToken)
	return token, ok
}
