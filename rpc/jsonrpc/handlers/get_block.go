package handlers

import (
	"context"
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/rpc/utils"
	"github.com/tinychain/tinychain/rpc/api"
)

type getBlockParams struct {
	Height uint64 `json:"height"`
	Hash   string `json:"hash"`
}

type getBlockResult struct {
	Block utils.Block `json:"block"`
}

type GetBlockHandler struct {
	api *api.ChainAPI
}

func (h GetBlockHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var p getBlockParams
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	hash := common.HexToHash(p.Hash)
	height := p.Height
	blk := h.api.GetBlock(hash, height)
	if blk == nil {
		return nil, &jsonrpc.Error{
			Code:    404,
			Message: "block not found",
		}
	}

	return getBlockResult{
		Block: *blk,
	}, nil
}

func (h GetBlockHandler) Name() string {
	return "GetBlock"
}

func (h GetBlockHandler) Params() interface{} {
	return getBlockParams{}
}

func (h GetBlockHandler) Result() interface{} {
	return getBlockResult{}
}
