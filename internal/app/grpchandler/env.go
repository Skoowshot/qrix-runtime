package grpchandler

import (
	"context"
	"errors"
	"io"
	"sync"

	envv1 "github.com/Skoowshot/qrix-runtime/gen/go/env/v1"
	"github.com/Skoowshot/qrix-runtime/internal/app/middleware"
	"github.com/Skoowshot/qrix-runtime/internal/app/service"
	"github.com/Skoowshot/qrix-runtime/internal/env"
	"github.com/Skoowshot/qrix-runtime/internal/rpc"
	"github.com/Skoowshot/qrix-runtime/internal/workspace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Environment struct {
	service *service.Environment
	envv1.UnimplementedEnvironmentServiceServer
}

func NewEnvironment(service *service.Environment) *Environment {
	return &Environment{
		service: service,
	}
}

func (e *Environment) Connect(ctx context.Context, req *envv1.ConnectRequest) (*envv1.ConnectResponse, error) {
	wsID := workspace.ID(req.WorkspaceId)
	name := env.Name(req.Name)
	metadata := env.Metadata(req.Metadata)

	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "missing name")
	}
	if len(metadata) == 0 {
		metadata = env.Metadata("{}")
	}
	if !metadata.IsValid() {
		return nil, status.Error(codes.InvalidArgument, "invalid metadata")
	}

	token, err := e.service.Connect(ctx, wsID, name, metadata)
	if err != nil {
		return nil, err
	}

	return &envv1.ConnectResponse{
		SessionToken: string(token),
	}, nil
}

func (e *Environment) OpenCall(stream grpc.BidiStreamingServer[envv1.OpenCallRequest, envv1.OpenCallResponse]) error {
	var wg sync.WaitGroup

	wsID, ok := middleware.WorkspaceFromContext(stream.Context())
	if !ok {
		return status.Error(codes.InvalidArgument, "missing workspace")
	}

	token, ok := env.SessionToken(""), true // todo header middleware

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	pipe, err := e.service.OpenCall(ctx, wsID, token)
	if err != nil {
		return err
	}
	defer close(pipe.In)

	// sender (out to grpc)
	wg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case frame, ok := <-pipe.Out:
				if !ok {
					cancel()
					return
				}

				f := envv1.CallFrame{
					Id:   uint64(frame.ID),
					Kind: nil, //todo map kind
				}
				if err := stream.Send(&envv1.OpenCallResponse{Frame: &f}); err != nil {
					cancel()
					return
				}
			}
		}
	})

	// receiver (grpc to in)
	for {
		req, err := stream.Recv()
		if err != nil {
			cancel()
			wg.Wait()
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		frame := req.GetFrame() //todo check if nil

		f := rpc.CallFrame{
			ID: rpc.ID(frame.Id),
			// todo map kind
		}

		select {
		case pipe.In <- f:
		case <-ctx.Done():
			wg.Wait()
			return nil
		}
	}
}

func (e *Environment) OpenStream(stream grpc.BidiStreamingServer[envv1.OpenStreamRequest, envv1.OpenStreamResponse]) error {
	return status.Error(codes.Unimplemented, "method OpenStream not implemented")
}
