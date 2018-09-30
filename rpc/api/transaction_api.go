package api

import (
	"tinychain/common"
	"tinychain/core"
	"tinychain/core/types"
	"tinychain/event"
	"tinychain/rpc/utils"
	"tinychain/tiny"
)

type TransactionAPI struct {
	tiny *tiny.Tiny
}

func (api *TransactionAPI) GetTransaction(hash common.Hash) (*utils.Transaction, common.Hash, uint64) {
	txMeta, err := api.tiny.DB().GetTxMeta(hash)
	if err != nil {
		return nil, common.Hash{}, 0
	}

	block := api.tiny.Chain().GetBlock(txMeta.Hash, txMeta.Height)
	if block != nil {
		return nil, common.Hash{}, 0
	}
	tx := block.Transactions[txMeta.TxIndex]
	return convertTransaction(tx), block.Hash(), block.Height()
}

func (api *TransactionAPI) SendTransaction(tx *types.Transaction) {
	ev := event.GetEventhub()
	go ev.Post(&core.NewTxEvent{tx})
}

func (api *TransactionAPI) GetReceipt(txHash common.Hash) *utils.Receipt {
	receipt, err := api.tiny.DB().GetReceipt(txHash)
	if err != nil {
		return nil
	}
	return convertReceipt(receipt)
}
