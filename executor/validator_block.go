package executor

import (
	"tinychain/core/types"
	"errors"
	"tinychain/core/state"
)

var (
	ErrTxRootNotEqual      = errors.New("txs root is not equal")
	ErrReceiptRootNotEqual = errors.New("receipts root is not equal")
	ErrStateRootNotEqual   = errors.New("state root is not equal")
)

type BlockValidatorImpl struct {
	chain Blockchain
}

func NewBlockValidator(chain Blockchain) *BlockValidatorImpl {
	return &BlockValidatorImpl{
		chain: chain,
	}
}

// Validate block header
// 1. Validate timestamp
// 2. Validate gasUsed and gasLimit
// 3. Validate parentHash and height
// 4. Validate extra data size is within bounds
// 5. Validate transactions and tx root
func (v *BlockValidatorImpl) ValidateHeader(block *types.Block) error {
	txRoot := block.Transactions.Hash()
	if txRoot != block.TxRoot() {
		return ErrTxRootNotEqual
	}
}

// Validate block txs
// 1. Validate receipts root hash
// 2. Validate state root
func (v *BlockValidatorImpl) ValidateState(block *types.Block, state *state.StateDB, receipts types.Receipts) error {
	receiptRoot := receipts.Hash()
	if receiptRoot != block.ReceiptsHash() {
		return ErrReceiptRootNotEqual
	}

	root, err := state.IntermediateRoot()
	if err != nil {
		return err
	}
	if root != block.StateRoot() {
		return ErrStateRootNotEqual
	}
	return nil
}
