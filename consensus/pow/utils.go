package pow

import (
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/core/types"
	"math/big"
	"encoding/binary"
	"math"
	"encoding/json"
	"time"
)

func computeHash(nonce uint64, header *types.Header) ([]byte, error) {
	var nonceBytes []byte
	binary.BigEndian.PutUint64(nonceBytes, nonce)
	hash := header.HashNoConsensus().Bytes()
	hash = append(hash, nonceBytes...)
	return common.Sha256(hash).Bytes(), nil
}

// computeNewDiff computed new difficulty with old diff and time duration
func computeNewDiff(currDiff uint64, curr *types.Block, old *types.Block) uint64 {
	duration := time.Duration(new(big.Int).Sub(curr.Time(), old.Time()).Int64())
	week := 7 * 24 * time.Hour
	if duration < week/4 {
		// if duration is lower than 1/4 week time, set to 1/4 week time
		duration = week / 4
	} else if duration > week*4 {
		// if duration is larger than 4 times of a week time, set to 4. week time.
		duration = week * 4
	}
	oldTarget := computeTarget(currDiff)
	total := new(big.Int).Mul(oldTarget, new(big.Int).SetInt64(int64(duration)))
	newTarget := total.Div(total, new(big.Int).SetInt64(int64(week)))
	maxTarget := new(big.Int).SetUint64(math.MaxUint64)
	// if larger than max target
	if newTarget.Cmp(maxTarget) == 1 {
		newTarget = maxTarget
	}

	return maxTarget.Div(maxTarget, newTarget).Uint64()
}

func computeTarget(difficulty uint64) *big.Int {
	maxTarget := new(big.Int).SetUint64(math.MaxUint64)
	return maxTarget.Div(maxTarget, new(big.Int).SetUint64(difficulty))
}

func decodeConsensusInfo(d []byte) (*consensusInfo, error) {
	ci := &consensusInfo{}
	err := json.Unmarshal(d, ci)
	if err != nil {
		return nil, err
	}
	return ci, nil
}
