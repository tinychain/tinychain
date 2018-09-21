package handlers

import (
	"context"
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	"tinychain/core/chain"
)

type getBlockHashParams struct {
	Height uint64
}

type getBlockHashResult struct {
	Hash string `json:"hash"`
}

type GetBlockHashHandler struct{}

func (h GetBlockHashHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (result interface{}, err *jsonrpc.Error) {
	var p getBlockParams

	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	// TODO
	hash := chain.GetHash(p.Height)

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
