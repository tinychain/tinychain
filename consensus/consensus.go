package consensus

import (
	"tinychain/consensus/dpos_bft"
	"tinychain/core/types"
	"tinychain/common"
	"tinychain/core/state"
	"tinychain/core"
	bpool "tinychain/common/blockpool"
	"github.com/libp2p/go-libp2p-peer"
)

type Engine interface {
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

func New(config *common.Config, chain *core.Blockchain, id peer.ID) (Engine, error) {
	log := common.GetLogger("consensus")
	return dpos_bft.New(config, log, chain, id, bpool.NewBlockPool(config, log, common.PROPOSE_BLOCK_MSG))
}
