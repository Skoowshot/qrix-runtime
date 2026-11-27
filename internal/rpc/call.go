package rpc

type CallFrameKind uint8

const (
	CallFrameRequest CallFrameKind = iota
	CallFrameResponse
)

type CallFrame struct {
	ID   ID
	Kind CallFrameKind
}

type CallPipe struct {
	In  chan<- CallFrame
	Out <-chan CallFrame
}
