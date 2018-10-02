package handlers

import (
	"context"
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	"github.com/tinychain/tinychain/rpc/api"
)

type getBlockHashParams struct {
	Height uint64
}

type getBlockHashResult struct {
	Hash string `json:"hash"`
}

type GetBlockHashHandler struct {
	api *api.ChainAPI
}

func (h GetBlockHashHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var p getBlockParams

	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	hash := h.api.GetBlockHash(p.Height)
	return getBlockHashResult{
		Hash: string(hash.Hex()),
	}, nil
}

func (h GetBlockHashHandler) Name() string {
	return "GetBlockByHash"
}

func (h GetBlockHashHandler) Params() interface{} {
	return getBlockHashParams{}
}

func (h GetBlockHashHandler) Result() interface{} {
	return getBlockHashResult{}
}
