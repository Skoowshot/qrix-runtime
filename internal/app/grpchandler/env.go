package grpchandler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
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

	token, ok := middleware.SessionTokenFromContext(stream.Context())
	if !ok {
		return status.Error(codes.InvalidArgument, "missing session token")
	}

	ctx, cancel := context.WithCancel(stream.Context())

	pipe, err := e.service.OpenCall(ctx, wsID, token)
	if err != nil {
		cancel()
		return err
	}

	defer close(pipe.In)
	defer wg.Wait()

	wg.Go(func() {
		defer cancel()
		sendCallFrames(ctx, pipe, stream)
	})

	defer cancel()
	if err := receiveCallFrames(ctx, pipe, stream); err != nil {
		return err
	}
	return nil
}

func (e *Environment) OpenStream(stream grpc.BidiStreamingServer[envv1.OpenStreamRequest, envv1.OpenStreamResponse]) error {
	return status.Error(codes.Unimplemented, "method OpenStream not implemented")
}

func sendCallFrames(ctx context.Context, pipe rpc.CallPipe, stream grpc.BidiStreamingServer[envv1.OpenCallRequest, envv1.OpenCallResponse]) {
	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-pipe.Out:
			if !ok {
				return
			}

			f, err := mapRpcCallFrame(frame)
			if err != nil {
				slog.ErrorContext(ctx, "failed to map outgoing call frame", "err", err)
				return
			}
			if err := stream.Send(&envv1.OpenCallResponse{Frame: f}); err != nil {
				slog.ErrorContext(ctx, "failed to send response", "err", err)
				return
			}
		}
	}
}

func receiveCallFrames(ctx context.Context, pipe rpc.CallPipe, stream grpc.BidiStreamingServer[envv1.OpenCallRequest, envv1.OpenCallResponse]) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		frame := req.GetFrame()

		f, err := mapProtoCallFrame(frame)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid frame: %v", err)
		}

		select {
		case pipe.In <- f:
		case <-ctx.Done():
			return nil
		}
	}
}

func mapProtoCallFrame(frame *envv1.CallFrame) (rpc.CallFrame, error) {
	if frame == nil {
		return rpc.CallFrame{}, fmt.Errorf("missing frame")
	}

	id := rpc.ID(frame.Id)

	switch k := frame.GetKind().(type) {
	case *envv1.CallFrame_Request:
		return rpc.NewCallFrameRequest(id), nil
	case *envv1.CallFrame_Response:
		return rpc.NewCallFrameResponse(id), nil
	case *envv1.CallFrame_Fail:
		fail := k.Fail
		rpcErr, err := mapProtoRpcError(fail.GetError())
		if err != nil {
			return rpc.CallFrame{}, err
		}
		return rpc.NewCallFrameFail(id, rpcErr), nil
	default:
		return rpc.CallFrame{}, fmt.Errorf("unknown frame kind")
	}
}

func mapRpcCallFrame(frame rpc.CallFrame) (*envv1.CallFrame, error) {
	f := &envv1.CallFrame{
		Id: uint64(frame.ID),
	}
	switch frame.Kind {
	case rpc.CallFrameRequest:
		f.Kind = &envv1.CallFrame_Request{Request: &envv1.CallRequest{}}
	case rpc.CallFrameResponse:
		f.Kind = &envv1.CallFrame_Response{Response: &envv1.CallResponse{}}
	case rpc.CallFrameFail:
		rpcErr, err := mapRpcError(frame.Error())
		if err != nil {
			return nil, err
		}

		f.Kind = &envv1.CallFrame_Fail{Fail: &envv1.CallFail{
			Error: rpcErr,
		}}
	default:
		return nil, fmt.Errorf("unknown frame kind")
	}

	return f, nil
}

func mapProtoRpcError(rpcErr *envv1.RpcError) (rpc.Error, error) {
	if rpcErr == nil {
		return rpc.Error{}, fmt.Errorf("missing error")
	}
	return rpc.Error{
		Code:    rpc.Code(rpcErr.Code),
		Message: rpcErr.Message,
	}, nil
}

func mapRpcError(rpcErr *rpc.Error) (*envv1.RpcError, error) {
	if rpcErr == nil {
		return nil, fmt.Errorf("missing error")
	}
	return &envv1.RpcError{
		Code:    uint32(rpcErr.Code),
		Message: rpcErr.Message,
	}, nil
}
