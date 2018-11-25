package consensus

import (
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/core/state"
	"github.com/tinychain/tinychain/core/types"
	"github.com/tinychain/tinychain/core/chain"
	"github.com/tinychain/tinychain/consensus/solo"
	"github.com/tinychain/tinychain/consensus/pow"
	"errors"
)

type Engine interface {
	Start() error
	Stop() error
	// protocol handled at p2p layer
	Protocols() []common.Protocol
	// Finalize a valid block
	Finalize(header *types.Header, state *state.StateDB, txs types.Transactions, receipts types.Receipts) (*types.Block, error)
}

func New(config *common.Config, state *state.StateDB, chain chain.Blockchain) (Engine, error) {
	engineName := config.GetString(common.EngineName)
	switch engineName {
	case common.SoloEngine:
		return solo.New(config, state, chain)
	case common.PowEngine:
		return pow.New(config, state, chain)
	default:
		return nil, errors.New("invalid consensus engine name")
	}
}
