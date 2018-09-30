package jsonrpc

import (
	"github.com/osamingo/jsonrpc"
	"net/http"
	"tinychain/rpc/api"
	"tinychain/rpc/jsonrpc/handlers"
	"tinychain/tiny"
)

type Handler interface {
	jsonrpc.Handler
	Name() string
	Params() interface{}
	Result() interface{}
}

func InitHandler(t *tiny.Tiny) []Handler {
	chainApi := &api.ChainAPI{t}
	txApi := &api.TransactionAPI{t}
	return []Handler{
		handlers.GetBlockHandler{chainApi},
		handlers.GetBlockHashHandler{chainApi},
		handlers.GetHeaderHandler{chainApi},
		handlers.GetTxHandler{txApi},
		handlers.GetReceiptHandler{txApi},
		handlers.SendTxHandler{txApi},
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
