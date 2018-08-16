package pow

import (
	"tinychain/core/types"
	"math/big"
	"errors"
)

var (
	errInvalidNonce = errors.New("consensus nonce invalid")
)

// csValidator verifies the consensus information of a new block
type csValidator struct {
	target *big.Int
}

func newCsValidator(difficulty uint64) *csValidator {
	return &csValidator{
		target: computeTarget(difficulty),
	}
}

func (cv *csValidator) Validate(block *types.Block) error {
	consensus := &consensusInfo{}
	err := consensus.Deserialize(block.ConsensusInfo())
	if err != nil {
		return err
	}

	tbits := consensus.Difficulty
	nonce := consensus.Nonce

	hash, err := computeHash(tbits, nonce, block)
	if err != nil {
		log.Errorf("failed to compute consensus hash, err:%s", err)
		return err
	}

	if cv.target.Cmp(new(big.Int).SetBytes(hash)) == 1 {
		return nil
	}
	return errInvalidNonce
}
