package jsonrpc

import (
	"github.com/osamingo/jsonrpc"
	"tinychain/rpc/jsonrpc/handlers"
)

type Handler interface {
	jsonrpc.Handler
	Name() string
	Params() interface{}
	Result() interface{}
}

func Handlers() []Handler {
	return []Handler{
		handlers.GetBlockHandler{},
		handlers.GetBlockHashHandler{},
	}
}

func StartJsonRPCServer() error {
	mr := jsonrpc.NewMethodRepository()

	for _, s := range Handlers() {
		mr.RegisterMethod(s.Name(), s, s.Params(), s.Result())
	}
}
