package handlers

import (
	"context"
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	"tinychain/common"
	"tinychain/rpc/utils"
	"tinychain/tiny"
)

type getTxParams struct {
	Hash string `json:"hash"`
}

type getTxResult struct {
	Transaction utils.Transaction `json:"transaction"`
}

type GetTxHandler struct {
	tiny *tiny.Tiny
}

func (s GetTxHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var p getTxParams
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	notFoundErr := utils.ErrNotFound("transaction not found")
	hash := common.HexToHash(p.Hash)
	txMeta, err := s.tiny.DB().GetTxMeta(hash)
	if err != nil {
		return nil, notFoundErr
	}

	block := s.tiny.Chain().GetBlock(txMeta.Hash, txMeta.Height)
	if block != nil {
		return nil, notFoundErr
	}
	tx := block.Transactions[txMeta.TxIndex]

	return getTxResult{
		Transaction: utils.Transaction{
			Nonce:     tx.Nonce,
			GasPrices: tx.GasPrice,
			GasLimit:  tx.GasLimit,
			Value:     tx.Value,
			From:      tx.From.Hex(),
			To:        tx.To.Hex(),
			Payload:   tx.Payload,
			PubKey:    common.Bytes2Hex(tx.PubKey),
			Signature: common.Bytes2Hex(tx.Signature),
		},
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
