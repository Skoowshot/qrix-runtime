package rpc

type StreamFrameKind uint8

const (
	StreamFrameFail StreamFrameKind = iota
	StreamFrameStart
	StreamFrameChunk
	StreamFrameEnd
)

type StreamFrame struct {
	ID   ID
	Kind StreamFrameKind

	error Error
}

func NewStreamFrameFail(id ID, err Error) StreamFrame {
	return StreamFrame{
		ID:    id,
		Kind:  StreamFrameFail,
		error: err,
	}
}

func NewStreamFrameStart(id ID) StreamFrame {
	return StreamFrame{
		ID:   id,
		Kind: StreamFrameStart,
	}
}

func NewStreamFrameChunk(id ID) StreamFrame {
	return StreamFrame{
		ID:   id,
		Kind: StreamFrameChunk,
	}
}

func NewStreamFrameEnd(id ID) StreamFrame {
	return StreamFrame{
		ID:   id,
		Kind: StreamFrameEnd,
	}
}

func (f StreamFrame) Error() *Error {
	if f.Kind != StreamFrameFail {
		return nil
	}
	return &f.error
}

type StreamPipe struct {
	In  chan<- StreamFrame
	Out <-chan StreamFrame
}
