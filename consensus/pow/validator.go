package pow

import (
	"tinychain/core/types"
)

// validator verifies the consensus information of a new block
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

	tbits := consensus.Difficulty
	nonce := consensus.Nonce

}
