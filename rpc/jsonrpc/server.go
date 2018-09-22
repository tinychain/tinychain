package jsonrpc

import (
	"github.com/osamingo/jsonrpc"
	"tinychain/rpc/jsonrpc/handlers"
	"net/http"
	"tinychain/tiny"
)

type Handler interface {
	jsonrpc.Handler
	Name() string
	Params() interface{}
	Result() interface{}
}

func InitHandler(t *tiny.Tiny) []Handler {
	return []Handler{
		handlers.GetBlockHandler{t},
		handlers.GetBlockHashHandler{t},
		handlers.GetHeaderHandler{t},
		handlers.GetTxHandler{t},
	}
}

func StartJsonRPCServer(t *tiny.Tiny) error {
	mr := jsonrpc.NewMethodRepository()

	for _, s := range InitHandler(t) {
		mr.RegisterMethod(s.Name(), s, s.Params(), s.Result())
	}

	http.Handle("/", mr)
	http.HandleFunc("/debug", mr.ServeDebug)

	if err := http.ListenAndServe(":8081", http.DefaultServeMux); err != nil {
		return err
	}
	return nil
}
