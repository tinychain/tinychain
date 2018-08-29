// Copyright (c) 2012, Suryandaru Triandana <syndtr@gmail.com>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"container/list"
	"sync"
	"unsafe"
)

// LBN, namely 'Latest Block Num', is used to manage and evict cache data
// which is set at the block num lower than given block num.
//
// LBN implements Cacher.
type lbnNode struct {
	n   *Node
	h   *Handle
	ban bool
}

// bnNode holds the block number and the lbnNode inserted at this number
type bnNode struct {
	blockNum uint64
	list     *list.List
}

func newBnNode(blockNum uint64) *bnNode {
	return &bnNode{
		blockNum: blockNum,
		list:     list.New(),
	}
}

type pos struct {
	bn *bnNode
	rn *list.Element // lbnNode, wrapped by list.Element
}

func (p *pos) getLBNNode() *lbnNode {
	return p.rn.Value.(*lbnNode)
}

type lbn struct {
	mu       sync.Mutex
	capacity int
	used     int

	list *list.List
}

func (r *lbn) reset() {
	r.list = list.New()
	r.used = 0
}

func (r *lbn) getBnNodeByNum(blockNum uint64) *bnNode {
	for rn := r.list.Front(); rn != nil; rn = rn.Next() {
		bn := rn.Value.(*bnNode)
		if bn.blockNum == blockNum {
			return bn
		}
	}

	bn := newBnNode(blockNum)
	r.list.PushFront(bn)
	return bn
}

func (r *lbn) Promote(n *Node) {
	var evicted []*lbnNode

	r.mu.Lock()
	if n.CacheData == nil {
		ln := &lbnNode{n: n, h: n.GetHandle()}
		r.used += n.Size()

		// FIXME: we do not consider the capacity in period 1
		//for r.used > r.capacity {
		//	rn := r.head.prev
		//	if rn == nil {
		//		panic("BUG: invalid LRU used or capacity counter")
		//	}
		//	rn.remove()
		//	rn.n.CacheData = nil
		//	r.used -= rn.n.Size()
		//	evicted = append(evicted, rn)
		//}
		bn := r.getBnNodeByNum(n.blockNum)
		n.CacheData = unsafe.Pointer(&pos{bn, bn.list.PushBack(ln)})
	} else {
		position := (*pos)(n.CacheData)
		ln := position.getLBNNode()
		if !ln.ban {
			position.bn.list.Remove(position.rn)
			bn := r.getBnNodeByNum(n.blockNum)
			n.CacheData = unsafe.Pointer(&pos{bn, bn.list.PushBack(ln)})
		}
	}
	r.mu.Unlock()

	for _, rn := range evicted {
		rn.h.Release()
	}
}

func (r *lbn) Ban(n *Node) {
	r.mu.Lock()
	if n.CacheData == nil {
		n.CacheData = unsafe.Pointer(&lbnNode{n: n, ban: true})
	} else {
		position := (*pos)(n.CacheData)
		ln := position.getLBNNode()
		if !ln.ban {
			position.bn.list.Remove(position.rn)
			ln.ban = true
			r.used -= ln.n.Size()
			r.mu.Unlock()

			ln.h.Release()
			ln.h = nil
			return
		}
	}
	r.mu.Unlock()
}

func (r *lbn) Evict(n *Node) {
	r.mu.Lock()
	position := (*pos)(n.CacheData)
	if position == nil {
		r.mu.Unlock()
		return
	}
	ln := position.getLBNNode()
	if ln.ban {
		r.mu.Unlock()
		return
	}
	position.bn.list.Remove(position.rn)
	n.CacheData = nil
	r.mu.Unlock()

	ln.h.Release()
}

func (r *lbn) EvictWithStrategy(strategy func(blockNum uint64) bool) int {
	var evicted []*lbnNode
	r.mu.Lock()

	for bn := r.list.Front(); bn != nil; {
		e := bn
		bn = bn.Next()
		bnNode := e.Value.(*bnNode)
		if strategy(bnNode.blockNum) {
			for en := bnNode.list.Front(); en != nil; en = en.Next() {
				ln := en.Value.(*lbnNode)
				ln.n.CacheData = nil
				r.used -= ln.n.Size()
				evicted = append(evicted, ln)
			}
			bnNode.list.Init()

			r.list.Remove(e)
		}
	}
	r.mu.Unlock()

	for _, ln := range evicted {
		ln.h.Release()
	}

	return len(evicted)
}

func (r *lbn) EvictAll() {
	var evicted []*lbnNode
	r.mu.Lock()
	for bn := r.list.Front(); bn != nil; bn = bn.Next() {
		bnNode := bn.Value.(*bnNode)
		for rn := bnNode.list.Front(); rn != nil; rn = rn.Next() {
			ln := rn.Value.(*lbnNode)
			ln.n.CacheData = nil
			evicted = append(evicted, ln)
		}
	}
	r.reset()
	r.mu.Unlock()

	for _, ln := range evicted {
		ln.h.Release()
	}
}

func (r *lbn) Close() error {
	r.EvictAll()
	return nil
}

func NewLBN() Cacher {
	r := &lbn{}
	r.reset()
	return r
}
