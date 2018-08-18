package pow

import (
	"tinychain/common"
	"tinychain/core/types"
	"math/big"
	"github.com/ethereum/go-ethereum/common/math"
)

func computeHash(difficulty uint64, nonce uint64, block *types.Block) ([]byte, error) {
	consensus := &consensusInfo{
		Difficulty: difficulty,
		Nonce:      nonce,
	}
	data, err := consensus.Serialize()
	if err != nil {
		return nil, err
	}
	clone := block.Clone()
	clone.Header.ConsensusInfo = data
	seed := clone.Hash()
	hash := common.Sha256(seed.Bytes())
	return hash.Bytes(), nil
}

func computeTarget(difficulty uint64) *big.Int {
	maxTarget := new(big.Int).SetUint64(math.MaxUint64)
	return maxTarget.Div(maxTarget, new(big.Int).SetUint64(difficulty))
}
