package pow

import (
	"tinychain/core/types"
	"math/big"
	"errors"
)

var (
	errInvalidDiff  = errors.New("difficulty invalid")
	errInvalidNonce = errors.New("consensus nonce invalid")
)

// csValidator verifies the consensus information of a new block
type csValidator struct {
	chain Blockchain
}

func newCsValidator(chain Blockchain) *csValidator {
	return &csValidator{chain: chain}
}

// Validate verify the consensus info in the block
// 1. Verify the difficulty is equal to the result calculated by that of parent block
// 2. Verify the nonce is valid or not
func (cv *csValidator) Validate(block *types.Block) error {
	consensus, err := decodeConsensusInfo(block.ConsensusInfo())
	if err != nil {
		return err
	}

	// Verify difficulty
	parent := cv.chain.GetBlockByHash(block.ParentHash())
	parentCons, err := decodeConsensusInfo(parent.ConsensusInfo())
	if err != nil {
		return err
	}
	if parentCons.Difficulty != consensus.Difficulty {
		old := cv.chain.GetBlockByHeight(block.Height() - blockGapForDiffAdjust)
		if computeNewDiff(parentCons.Difficulty, block, old) != consensus.Difficulty {
			return errInvalidDiff
		}
	}

	target := computeTarget(consensus.Difficulty)
	nonce := consensus.Nonce

	// Verify proof-of-work result and nonce
	hash, err := computeHash(nonce, block.Header)
	if err != nil {
		log.Errorf("failed to compute consensus hash, err:%s", err)
		return err
	}

	if target.Cmp(new(big.Int).SetBytes(hash)) == 1 {
		return nil
	}
	return errInvalidNonce
}
