package executor

import (
	"tinychain/core/types"
	"errors"
	"tinychain/core/state"
	"tinychain/common"
)

var (
	errTxRootNotEqual      = errors.New("txs root is not equal")
	errReceiptRootNotEqual = errors.New("receipts root is not equal")
	errStateRootNotEqual   = errors.New("state root is not equal")
	errGasUsedOverflow     = errors.New("gas used is larger than gas limit")
	errParentHashNotMatch  = errors.New("parent hash is not match")
	errExtraDataOverflow   = errors.New("extra data is too long")
)

type BlockValidatorImpl struct {
	maxExtraLength uint64
	chain          Blockchain
}

func NewBlockValidator(config *common.Config, chain Blockchain) *BlockValidatorImpl {
	return &BlockValidatorImpl{
		maxExtraLength: uint64(config.GetInt64(common.MAX_EXTRA_LENGTH)),
		chain:          chain,
	}
}

// Validate block header
// 1. Validate timestamp
// 2. Validate gasUsed and gasLimit
// 3. Validate parentHash and height
// 4. Validate extra data size is within bounds
// 5. Validate transactions and tx root
func (v *BlockValidatorImpl) ValidateHeader(block *types.Block) error {
	//  TODO Check timestamp


	if block.GasUsed() > block.GasLimit() {
		return errGasUsedOverflow
	}

	last := v.chain.LastBlock().Hash()
	if last != block.ParentHash() {
		return errParentHashNotMatch
	}

	if uint64(len(block.Extra())) > v.maxExtraLength {
		return errExtraDataOverflow
	}

	txRoot := block.Transactions.Hash()
	if txRoot != block.TxRoot() {
		return errTxRootNotEqual
	}
	return nil
}

// Validate block txs
// 1. Validate receipts root hash
// 2. Validate state root
func (v *BlockValidatorImpl) ValidateState(block *types.Block, state *state.StateDB, receipts types.Receipts) error {
	receiptRoot := receipts.Hash()
	if receiptRoot != block.ReceiptsHash() {
		return errReceiptRootNotEqual
	}

	root, err := state.IntermediateRoot()
	if err != nil {
		return err
	}
	if root != block.StateRoot() {
		return errStateRootNotEqual
	}
	return nil
}
