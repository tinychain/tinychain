package handlers

import (
	"context"
	"math/big"

	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/core/types"
	"github.com/tinychain/tinychain/internal/api"
)

type sendTxParams struct {
	Nonce     uint64   `json:"nonce"`
	GasPrice  uint64   `json:"gas_price"`
	GasLimit  uint64   `json:"gas_limit"`
	Value     *big.Int `json:"value"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Payload   []byte   `json:"payload"`
	PubKey    string   `json:"pub_key"`
	Signature string   `json:"signature"`
}

type sendTxResult struct {
	TxHash string `json:"tx_hash"`
}

type SendTxHandler struct {
	api *api.TransactionAPI
}

func (s SendTxHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var p sendTxParams
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	tx := types.NewTransaction(
		p.Nonce,
		p.GasPrice,
		p.GasLimit,
		p.Value,
		p.Payload,
		common.HexToAddress(p.From),
		common.HexToAddress(p.To),
	)

	tx.PubKey = common.Hex2Bytes(p.PubKey)
	tx.Signature = common.Hex2Bytes(p.Signature)

	s.api.SendTransaction(tx)
	return sendTxResult{

	}, nil
}

func (h SendTxHandler) Name() string {
	return "SendTransaction"
}

func (h SendTxHandler) Params() interface{} {
	return sendTxParams{}
}

func (h SendTxHandler) Result() interface{} {
	return sendTxResult{}
}
