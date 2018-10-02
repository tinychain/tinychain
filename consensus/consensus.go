package consensus

import (
	"github.com/tinychain/tinychain/core/state"
	"github.com/tinychain/tinychain/core/types"
	"github.com/tinychain/tinychain/p2p"
)

type Engine interface {
	Start() error
	Stop() error
	// protocol handled at p2p layer
	Protocols() []p2p.Protocol
	// Finalize a valid block
	Finalize(header *types.Header, state *state.StateDB, txs types.Transactions, receipts types.Receipts) (*types.Block, error)
}

type BlockValidator interface {
	ValidateHeader(header *types.Header) error
	ValidateHeaders(headers []*types.Header) (chan struct{}, chan error)
	ValidateBody(b *types.Block) error
	ValidateState(b *types.Block, state *state.StateDB, receipts types.Receipts) error
}

type TxValidator interface {
	ValidateTxs(types.Transactions) (types.Transactions, types.Transactions)
	ValidateTx(*types.Transaction) error
}

type Blockchain interface {
	LastBlock() *types.Block      // Last block in memory
	LastFinalBlock() *types.Block // Last irreversible block
	GetBlockByHeight(height uint64) *types.Block
}

//func New(config *common.Config, engineName string, state *state.StateDB, chain *chain.Blockchain, id peer.ID, blValidator *executor.BlockValidator, txValidator *executor.TxValidator) (Engine, error) {
//	switch engineName {
//	case common.SoloEngine:
//		return solo.New(config, state, chain, blValidator, txValidator)
//	case common.PowEngine:
//		return pow.New(config, state, chain, blValidator, txValidator)
//	case common.VrfBftEngine:
//		return vrf.New(config, state, chain, id, blValidator)
//	case common.DposBftEngine:
//		return dpos_bft.New(config, state, chain, id, blValidator)
//	default:
//		return nil, errors.New("invalid consensus engine name")
//	}
//}
