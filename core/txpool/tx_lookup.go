package txpool

import (
	"tinychain/common"
	"sync"
)

type txLookup struct {
	mu  sync.RWMutex
	all map[common.Hash]struct{}
}

func newTxLookup() *txLookup {
	return &txLookup{}
}

func (tl *txLookup) Len() uint64 {
	tl.mu.RLock()
	defer tl.mu.RUnlock()
	return uint64(len(tl.all))
}

func (tl *txLookup) Get(hash common.Hash) bool {
	tl.mu.RLock()
	defer tl.mu.RUnlock()
	_, exist := tl.all[hash]
	return exist
}

func (tl *txLookup) Add(hash common.Hash) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.all[hash] = struct{}{}
}

func (tl *txLookup) Del(hash common.Hash) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	delete()
}
