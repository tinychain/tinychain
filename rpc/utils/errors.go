package utils

import "github.com/osamingo/jsonrpc"

var (
	errNotFoundCode jsonrpc.ErrorCode = 404
)

func ErrNotFound(message string) *jsonrpc.Error {
	return &jsonrpc.Error{
		Code:    errNotFoundCode,
		Message: message,
	}
}
