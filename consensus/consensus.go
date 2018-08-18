package consensus

import (
	"tinychain/core/types"
	"tinychain/common"
	"tinychain/core/state"
	"tinychain/core"
	"github.com/libp2p/go-libp2p-peer"
	"tinychain/p2p"
	"tinychain/consensus/dpos_bft"
	"tinychain/executor"
)

type Engine interface {
	Start() error
	Stop() error
	Protocols() []p2p.Protocol
	Finalize(header *types.Header, state *state.StateDB, txs types.Transactions, receipts types.Receipts) (*types.Block, error)
}

type TxPool interface {
	// AddRemotes adds remote transactions to queue tx list
	AddRemotes(txs types.Transactions) error

	// Pending returns all valid and processable transactions
	Pending() map[common.Address]types.Transactions
}

func New(config *common.Config, state *state.StateDB, chain *core.Blockchain, id peer.ID, validator executor.BlockValidator) (Engine, error) {
	return dpos_bft.New(config, state, chain, id, validator)
}
