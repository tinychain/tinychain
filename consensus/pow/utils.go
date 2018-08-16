package pow

import (
	"tinychain/common"
	"tinychain/core/types"
	"math/big"
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
	target := new(big.Int).SetUint64(1)
	return target.Lsh(target, uint(256-difficulty))
}
