package rpc

type Code uint32

const (
	CodeOK Code = iota
	CodeUnknown
	CodeInvalidArgument
	CodeNotFound
	CodeAlreadyExists
	CodePermissionDenied
	CodeCanceled
	CodeDeadlineExceeded
	CodeUnimplemented
	CodeInternal
)

type Error struct {
	Code    Code
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
