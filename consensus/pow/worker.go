package pow

import (
	"math/big"
	"tinychain/core/types"
)

type worker struct {
	block      *types.Block
	difficulty uint64
	target     *big.Int // the computing hash should be lower than this target
	minBound   uint64   // computing nonce from
	maxBound   uint64   // computing nonce to
}

func newWorker(difficulty uint64, minBound uint64, maxBound uint64, block *types.Block) *worker {
	return &worker{
		difficulty: difficulty,
		block:      block,
		target:     computeTarget(difficulty),
		minBound:   minBound,
		maxBound:   maxBound,
	}
}

// Run starts a computing task to find a valid nonce
func (w *worker) Run(foundChan *nonceChan) {
	var result big.Int

	nonce := w.minBound
	for nonce <= w.maxBound {
		select {
		case <-foundChan.ch:
			return
		default:
		}
		hash, err := computeHash(nonce, w.difficulty, w.block)
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
