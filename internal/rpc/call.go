package rpc

type CallFrameKind uint8

const (
	CallFrameFail CallFrameKind = iota
	CallFrameRequest
	CallFrameResponse
)

type CallFrame struct {
	ID   ID
	Kind CallFrameKind

	error Error
}

func NewCallFrameFail(id ID, err Error) CallFrame {
	return CallFrame{
		ID:    id,
		Kind:  CallFrameFail,
		error: err,
	}
}

func NewCallFrameRequest(id ID) CallFrame {
	return CallFrame{
		ID:   id,
		Kind: CallFrameRequest,
	}
}

func NewCallFrameResponse(id ID) CallFrame {
	return CallFrame{
		ID:   id,
		Kind: CallFrameResponse,
	}
}

func (f CallFrame) Error() *Error {
	if f.Kind != CallFrameFail {
		return nil
	}
	return &f.error
}

type CallPipe struct {
	In  chan<- CallFrame
	Out <-chan CallFrame
}
