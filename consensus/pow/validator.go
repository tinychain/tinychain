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
}

func newCsValidator() *csValidator {
	return &csValidator{}
}

func (cv *csValidator) Validate(block *types.Block) error {
	consensus := &consensusInfo{}
	err := consensus.Deserialize(block.ConsensusInfo())
	if err != nil {
		return err
	}

	header := &types.Header{
		ParentHash: block.ParentHash(),
		Height:     block.Height(),
		Coinbase:   block.Coinbase(),
		Time:       block.Time(),
		GasLimit:   block.GasLimit(),
	}

	target := computeTarget(consensus.Difficulty)
	nonce := consensus.Nonce

	hash, err := computeHash(nonce, header)
	if err != nil {
		log.Errorf("failed to compute consensus hash, err:%s", err)
		return err
	}

	if target.Cmp(new(big.Int).SetBytes(hash)) == 1 {
		return nil
	}
	return errInvalidNonce
}
