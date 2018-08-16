package executor

import (
	"tinychain/core/types"
	"tinychain/core/state"
)

type TxValidator interface {
	ValidateTxs(txs types.Transactions) (types.Transactions, types.Transactions)
	ValidateTx(tx *types.Transaction) error
}

type BlockValidator interface {
	ValidateHeader(block *types.Block) error
	ValidateState(block *types.Block, state *state.StateDB, receipts types.Receipts) error
}