package dpos

import (
	"tinychain/core/types"
	"tinychain/core/state"
)

type DposEngine struct {
}

func NewDpos() *DposEngine {
	return &DposEngine{}
}

func (dpos *DposEngine) Name() string {
	return "TinyDPoS"
}

func (dpos *DposEngine) Start() error {
	return nil
}

func (dpos *DposEngine) Stop() error {
	return nil
}

func (dpos *DposEngine) Finalize(header *types.Header, state *state.StateDB, txs types.Transactions, receipts types.Receipts) (*types.Block, error) {
	root, err := state.IntermediateRoot()
	if err != nil {
		return nil, err
	}
	header.StateRoot = root
	header.ReceiptsHash = receipts.Hash()

	header.TxRoot = txs.Hash()
	return types.NewBlock(header, txs), nil
}
