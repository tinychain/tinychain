package txpool

import (
	"github.com/tinychain/tinychain/core/types"
	"sort"
	"math/big"
	"sync"
)

type txList struct {
	mu    sync.RWMutex
	txs   map[uint64]*types.Transaction
	cache types.Transactions
}

func newTxList() *txList {
	return &txList{}
}

func (list *txList) get(nonce uint64) *types.Transaction {
	list.mu.RLock()
	defer list.mu.RUnlock()
	return list.txs[nonce]
}

// Add adds a new transaction to the list, returning whether the
// transaction was accepted, and if yes, any previous transaction it replaced.
//
// PriceBump is the percent number
func (list *txList) add(tx *types.Transaction, priceBump int) (bool, *types.Transaction) {
	canInsert, old := list.CanInsert(tx, priceBump)
	if !canInsert {
		return false, nil
	}
	list.Put(tx)
	return true, old
}

func (list *txList) Put(tx *types.Transaction) {
	list.mu.Lock()
	defer list.mu.Unlock()
	list.txs[tx.Nonce] = tx
	list.cache = nil
}

func (list *txList) CanInsert(tx *types.Transaction, priceBump int) (bool, *types.Transaction) {
	var old *types.Transaction
	if old := list.get(tx.Nonce); old != nil {
		// Replacement strategy. Temporary design
		boundGas := old.GasLimit * uint64(100+priceBump) / 100
		if boundGas > tx.GasLimit {
			return false, nil
		}
	}
	return true, old
}

// Del deletes a transaction from the list
func (list *txList) Del(nonce uint64) {
	if old := list.get(nonce); old == nil {
		return
	}

	delete(list.txs, nonce)
	list.cache = nil
}

func (list *txList) Len() uint64 {
	list.mu.RLock()
	defer list.mu.RUnlock()
	return uint64(len(list.txs))
}

// All creates a nonce-sorted slice of current transaction list,
// and the result will be cache in case any modifications are made
func (list *txList) All() types.Transactions {
	if list.cache == nil {
		for _, tx := range list.txs {
			list.cache = append(list.cache, tx)
		}
		sort.Sort(types.NonceSortedList(list.cache))
	}

	results := make(types.Transactions, len(list.cache))
	copy(results, list.cache)

	return results
}

// Filter filters all transactions which make filter func true and false, and
// removes unmatching transactions from the list
func (list *txList) filter(filter func(tx *types.Transaction) bool) (types.Transactions, types.Transactions) {
	var (
		match   types.Transactions
		unmatch types.Transactions
	)
	list.mu.Lock()
	defer list.mu.Unlock()
	for _, tx := range list.txs {
		if filter(tx) {
			match = append(match, tx)
		} else {
			unmatch = append(unmatch, tx)
		}
	}

	for _, tx := range unmatch {
		delete(list.txs, tx.Nonce)
	}

	list.cache = nil
	return match, unmatch
}

// Ready retrieves a sequentially increasing list of transactions starting at the
// provided nonce that is ready for processing. The returned transactions will be
// removed from the list.
func (list *txList) Ready(start uint64) types.Transactions {
	var (
		results types.Transactions
		nonce   = start
	)
	list.mu.Lock()
	defer list.mu.Unlock()
	for {
		if tx, exist := list.txs[nonce]; exist {
			results = append(results, tx)
			nonce++
		} else {
			break
		}
	}
	for _, tx := range results {
		delete(list.txs, tx.Nonce)
	}
	list.cache = nil
	return results
}

// Forget drops all transactions whose nonce is lower than bound.
// Every removed transaction is returned.
func (list *txList) Forget(bound uint64) types.Transactions {
	_, drops := list.filter(func(tx *types.Transaction) bool {
		return tx.Nonce >= bound
	})
	return drops
}

// Release drops all transactions whose cost is over balance.
// Every removed transaction is returned.
func (list *txList) Release(balance *big.Int) types.Transactions {
	_, drops := list.filter(func(tx *types.Transaction) bool {
		return tx.Cost().Cmp(balance) <= 0
	})
	return drops
}