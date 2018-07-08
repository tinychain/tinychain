package consensus

import (
	"tinychain/consensus/dpos"
	"tinychain/core/types"
	"tinychain/common"
	"tinychain/core/state"
)

type Engine interface {
	Name() string
	Start() error
	Stop() error
	Finalize(header *types.Header, state *state.StateDB, txs types.Transactions, receipts types.Receipts) (*types.Block, error)
}

type TxPool interface {
	// AddRemotes adds remote transactions to queue tx list
	AddRemotes(txs types.Transactions) error

	// Pending returns all valid and processable transactions
	Pending() map[common.Address]types.Transactions
}

func New() Engine {
	return dpos.NewDpos()
}
