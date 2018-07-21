package executor

import (
	"tinychain/core/types"
)

type TxValidator interface {
	ValidateTxs(txs types.Transactions) (types.Transactions, types.Transactions)
	ValidateTx(tx *types.Transaction) error
}

type StateValidator interface {
	Process(tx types.Transactions, receipts types.Receipts) (valid types.Receipts, invalid types.Receipts)
}

type BlockValidator interface {
	ValidateHeader(block *types.Block) error
	ValidateState(block *types.Block, receipts types.Receipts) error
}
