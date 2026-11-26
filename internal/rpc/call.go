package rpc

type CallFrame struct {
	ID ID
}

type CallPipe struct {
	In  chan<- CallFrame
	Out <-chan CallFrame
}
