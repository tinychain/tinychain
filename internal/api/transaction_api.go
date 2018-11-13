package api

import (
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/core"
	"github.com/tinychain/tinychain/core/types"
	"github.com/tinychain/tinychain/event"
	"github.com/tinychain/tinychain/rpc/utils"
	"github.com/tinychain/tinychain/tiny"
	"github.com/tinychain/tinychain/core/executor"
	"github.com/tinychain/tinychain/core/state"
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

// Call executes the given transaction on the state for the given block height.
// It doesn't make and changes in the state/blockchain and is useful to execute and retrieve values.
func (api *TransactionAPI) Call(tx *types.Transaction) ([]byte, error) {
	chain := api.tiny.Chain()
	conf := api.tiny.Config()
	statedb, err := state.New(api.tiny.RawDB(), chain.LastBlock().StateRoot().Bytes())
	if err != nil {
		return nil, err
	}
	vmenv := executor.NewVM(conf, tx, chain.LastBlock().Header, chain, &tx.From, statedb)
	ret, _, _, err := executor.ApplyTx(vmenv, tx)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (api *TransactionAPI) GetReceipt(txHash common.Hash) *utils.Receipt {
	receipt, err := api.tiny.DB().GetReceipt(txHash)
	if err != nil {
		return nil
	}
	return convertReceipt(receipt)
}
