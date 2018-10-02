package executor

import (
	"errors"
	"runtime"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/core/chain"
	"github.com/tinychain/tinychain/core/state"
	"github.com/tinychain/tinychain/core/types"
)

var (
	errTimestampInvalid     = errors.New("timestamp of the block should be larger than that of parent block")
	errTxRootNotEqual       = errors.New("txs root is not equal")
	errReceiptRootNotEqual  = errors.New("receipts root is not equal")
	errStateRootNotEqual    = errors.New("state root is not equal")
	errGasUsedOverflow      = errors.New("gas used is larger than gas limit")
	errParentHeightNotMatch = errors.New("parent height mismatch")
	errExtraDataOverflow    = errors.New("extra data is too long")
	errBlockExist           = errors.New("block already existed in db")
	errParentNotExist       = errors.New("parent not exist")
)

type BlockValidator struct {
	maxExtraLength uint64
	chain          *chain.Blockchain
}

func NewBlockValidator(config *common.Config, chain *chain.Blockchain) *BlockValidator {
	return &BlockValidator{
		maxExtraLength: uint64(config.GetInt64(common.MAX_EXTRA_LENGTH)),
		chain:          chain,
	}
}

// ValidateHeader validate headers, called by outer obj
// 1. check header already exist or not
// 2. check parent exist or not
// 3. check detail header info
func (v *BlockValidator) ValidateHeader(header *types.Header) error {
	//  TODO Check timestamp
	if v.chain.GetHeader(header.Hash(), header.Height) != nil {
		return nil
	}
	parent := v.chain.GetHeaderByHash(header.ParentHash)
	if parent == nil {
		if header.Height != 0 {
			return nil
		}
		return errParentNotExist
	}
	return v.validateHeader(parent, header)
}

// ValidateHeaders validate headers in batch, used by sync-chain processing.
// It returns two channels, one is to abort the validate process, and the other is to store errors。
func (v *BlockValidator) ValidateHeaders(headers []*types.Header) (chan struct{}, chan error) {
	workers := runtime.GOMAXPROCS(runtime.NumCPU())
	if workers > len(headers) {
		workers = len(headers)
	}

	var (
		abort     = make(chan struct{})
		inputs    = make(chan int)
		done      = make(chan int, len(headers))
		errs      = make([]error, len(headers))
		errorsOut = make(chan error, len(headers))
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				var parent *types.Header
				header := headers[index]
				if index == 0 {
					parent = v.chain.GetHeader(header.ParentHash, header.Height)
				} else {
					parent = headers[index-1]
				}
				errs[index] = v.validateHeader(parent, header)
				done <- index
			}
		}()
	}

	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(headers))
		)
		for {
			select {
			case inputs <- in:
				if in++; in == len(headers) {
					// stop sending to workers
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errs[out]
					if out == len(headers)-1 {
						return
					}
				}
			}
		}
	}()
	return abort, errorsOut
}

// Validate block header
// 1. Validate timestamp
// 2. Validate gasUsed and gasLimit
// 3. Validate parentHash and height
// 4. Validate extra data size is within bounds
func (v *BlockValidator) validateHeader(parent, header *types.Header) error {
	if header.Time.Cmp(parent.Time) <= 0 {
		return errTimestampInvalid
	}

	if header.GasUsed > header.GasLimit {
		return errGasUsedOverflow
	}

	if header.Height != parent.Height+1 {
		return errParentHeightNotMatch
	}

	if uint64(len(header.Extra)) > v.maxExtraLength {
		return errExtraDataOverflow
	}
	return nil
}

// Validate block body
// 1. Check block already exist in db or not
// 1. Check transactions match tx root in header or not
func (v *BlockValidator) ValidateBody(block *types.Block) error {
	if old := v.chain.GetBlockByHash(block.Hash()); old != nil {
		return errBlockExist
	}
	txRoot := block.Transactions.Hash()
	if txRoot != block.TxRoot() {
		return errTxRootNotEqual
	}
	return nil
}

// Validate block txs
// 1. Validate receipts root hash
// 2. Validate Bloom filter
// 3. Validate state root
func (v *BlockValidator) ValidateState(block *types.Block, state *state.StateDB, receipts types.Receipts) error {
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
