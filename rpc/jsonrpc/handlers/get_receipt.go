package handlers

import (
	"context"
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	"tinychain/common"
	"tinychain/rpc/utils"
	"tinychain/tiny"
)

type getReceiptParams struct {
	Height uint64 `json:"block_height"`
	Hash   string `json:"block_hash"`
	TxHash string `json:"tx_hash"`
}

type getReceiptResult struct {
	Receipt utils.Receipt
}

type GetReceiptHandler struct {
	tiny *tiny.Tiny
}

func (h GetReceiptHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var p getReceiptParams
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	hash := common.HexToHash(p.Hash)
	txHash := common.HexToHash(p.TxHash)
	receipts, err := h.tiny.DB().GetReceipts(p.Height, hash)
	if err != nil {
		return nil, utils.ErrNotFound("receipts not found")
	}
	for _, rp := range receipts {
		if txHash == rp.TxHash {
			receipt := utils.Receipt{
				PostState:       rp.PostState.Hex(),
				Status:          rp.Status,
				TxHash:          rp.TxHash.Hex(),
				ContractAddress: rp.ContractAddress.Hex(),
				GasUsed:         rp.GasUsed,
			}

			for _, log := range rp.Logs {
				l := &utils.Log{
					Address:     log.Address.Hex(),
					Data:        log.Data,
					BlockHeight: log.BlockHeight,
					TxHash:      log.TxHash.Hex(),
					TxIndex:     log.TxIndex,
					BlockHash:   log.BlockHash.Hex(),
					Index:       log.Index,
					Removed:     log.Removed,
				}

				for _, topic := range log.Topics {
					l.Topics = append(l.Topics, topic.Hex())
				}

				receipt.Logs = append(receipt.Logs, l)
			}

			return getReceiptResult{
				Receipt: receipt,
			}, nil
		}
	}
	return nil, utils.ErrNotFound("receipt not found")
}

func (h GetReceiptHandler) Name() string {
	return "GetReceipt"
}

func (h GetReceiptHandler) Params() interface{} {
	return getReceiptParams{}
}

func (h GetReceiptHandler) Result() interface{} {
	return getReceiptResult{}
}
