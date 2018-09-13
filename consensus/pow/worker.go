package pow

import (
	"math/big"
	"tinychain/core/types"
)

type worker struct {
	header     *types.Header
	difficulty uint64
	target     *big.Int // the computing hash should be lower than this target
	minBound   uint64   // computing nonce from
	maxBound   uint64   // computing nonce to
}

func newWorker(difficulty uint64, minBound uint64, maxBound uint64, header *types.Header) *worker {
	return &worker{
		difficulty: difficulty,
		header:     header,
		target:     computeTarget(difficulty),
		minBound:   minBound,
		maxBound:   maxBound,
	}
}

// Run starts a computing task to find a valid nonce
func (w *worker) run(found chan uint64, abort chan struct{}) {
	var result big.Int

	nonce := w.minBound
	for nonce <= w.maxBound {
		select {
		case <-abort:
			return
		default:
			hash, err := computeHash(nonce, w.header)
			if err != nil {
				return
			}
			result.SetBytes(hash)
			if result.Cmp(w.target) == -1 {
				break
			}
			nonce++
		}
	}
	found <- nonce
}
