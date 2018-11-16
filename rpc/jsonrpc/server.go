package jsonrpc

import (
	"net/http"

	"github.com/osamingo/jsonrpc"
	"github.com/tinychain/tinychain/rpc/api"
	"github.com/tinychain/tinychain/rpc/jsonrpc/handlers"
	"github.com/tinychain/tinychain/tiny"
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

func Start(t *tiny.Tiny) error {
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
