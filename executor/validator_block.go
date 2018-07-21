package executor

import (
	"tinychain/core/types"
	"errors"
)

var (
	ErrTxRootNotEqual      = errors.New("txs root is not equal")
	ErrReceiptRootNotEqual = errors.New("receipts root is not equal")
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
func (v *BlockValidatorImpl) ValidateHeader(block *types.Block) error {

}

// Validate block txs
// 1. Validate txs root hash
// 2. Validate receipts root hash
func (v *BlockValidatorImpl) ValidateState(block *types.Block, receipts types.Receipts) error {
	txRoot := block.Transactions.Hash()
	if txRoot != block.TxRoot() {
		return ErrTxRootNotEqual
	}

	receiptRoot := receipts.Hash()
	if receiptRoot != block.ReceiptsHash() {
		return ErrReceiptRootNotEqual
	}

	return nil
}
