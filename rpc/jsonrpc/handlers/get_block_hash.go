package handlers

import (
	"context"
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	"tinychain/tiny"
	"tinychain/rpc/utils"
)

type getBlockHashParams struct {
	Height uint64
}

type getBlockHashResult struct {
	Hash string `json:"hash"`
}

type GetBlockHashHandler struct {
	tiny *tiny.Tiny
}

func (h GetBlockHashHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var p getBlockParams

	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	// TODO
	hash := h.tiny.Chain.GetHash(p.Height)
	if hash.Nil() {
		return nil, utils.ErrNotFound("block hash not found")
	}

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
