package consensus

import (
	"github.com/libp2p/go-libp2p-peer"
	"github.com/pkg/errors"
	"tinychain/common"
	"tinychain/consensus/dpos_bft"
	"tinychain/consensus/pow"
	"tinychain/consensus/solo"
	vrf "tinychain/consensus/vrf_bft"
	"tinychain/core/chain"
	"tinychain/core/executor"
	"tinychain/core/state"
	"tinychain/core/types"
	"tinychain/p2p"
)

type Engine interface {
	Start() error
	Stop() error
	Protocols() []p2p.Protocol
	Finalize(header *types.Header, state *state.StateDB, txs types.Transactions, receipts types.Receipts) (*types.Block, error)
}

func New(config *common.Config, engineName string, state *state.StateDB, chain *chain.Blockchain, id peer.ID, blValidator *executor.BlockValidator, txValidator *executor.TxValidator) (Engine, error) {
	switch engineName {
	case common.SoloEngine:
		return solo.New(config, state, chain, blValidator, txValidator)
	case common.PowEngine:
		return pow.New(config, state, chain, blValidator, txValidator)
	case common.VrfBftEngine:
		return vrf.New(config, state, chain, id, blValidator)
	case common.DposBftEngine:
		return dpos_bft.New(config, state, chain, id, blValidator)
	default:
		return nil, errors.New("invalid consensus engine name")
	}
}
