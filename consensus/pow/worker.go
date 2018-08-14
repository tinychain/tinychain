package pow

import (
	"math/big"
	"tinychain/core/types"
	"tinychain/common"
)

type worker struct {
	block      *types.Block
	difficulty uint64
	target     *big.Int // the computing hash should be lower than this target
	minBound   uint64   // computing nonce from
	maxBound   uint64   // computing nonce to
}

func newWorker(difficulty uint64, minBound uint64, maxBound uint64, block *types.Block) *worker {
	target := new(big.Int).SetUint64(1)
	target.Lsh(target, uint(256-difficulty))
	return &worker{
		difficulty: difficulty,
		block:      block,
		target:     target,
		minBound:   minBound,
		maxBound:   maxBound,
	}
}

// Run starts a computing task to find a valid nonce
func (w *worker) Run(foundChan *nonceChan) {
	var result big.Int
	// TODO
	nonce := w.minBound
	for nonce <= w.maxBound {
		select {
		case <-foundChan.ch:
			return
		default:
		}
		hash, err := w.computeHash(nonce)
		if err != nil {
			return
		}
		result.SetBytes(hash)
		if result.Cmp(w.target) == -1 {
			break
		}
		nonce++
	}
	foundChan.post(nonce)
}

func (w *worker) computeHash(nonce uint64) ([]byte, error) {
	consensus := &consensusInfo{
		Difficulty: w.difficulty,
		Nonce:      nonce,
	}
	data, err := consensus.Serialize()
	if err != nil {
		return nil, err
	}
	w.block.Header.ConsensusInfo = data
	first := w.block.Hash()
	hash := common.Sha256(first.Bytes())
	return hash.Bytes(), nil
}
