package handlers

import (
	"context"
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	"tinychain/common"
	"tinychain/rpc/api"
	"tinychain/rpc/utils"
)

type getTxParams struct {
	Hash string `json:"hash"`
}

type getTxResult struct {
	Transaction utils.Transaction `json:"transaction"`
	BlockHash   string            `json:"block_hash"`
	BlockHeight uint64            `json:"block_height"`
}

type GetTxHandler struct {
	api *api.TransactionAPI
}

func (s GetTxHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var p getTxParams
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	hash := common.HexToHash(p.Hash)
	tx, hash, height := s.api.GetTransaction(hash)
	if tx == nil {
		return nil, utils.ErrNotFound("transaction not found")
	}
	return getTxResult{
		Transaction: *tx,
		BlockHash:   hash.Hex(),
		BlockHeight: height,
	}, nil

}

func (s GetTxHandler) Name() string {
	return "GetTransaction"
}

func (s GetTxHandler) Params() interface{} {
	return getTxParams{}
}

func (s GetTxHandler) Result() interface{} {
	return getTxResult{}
}
