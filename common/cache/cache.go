// Copyright (c) 2012, Suryandaru Triandana <syndtr@gmail.com>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package cache provides interface and implementation of a cache algorithms.
package cache

import (
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Cacher provides interface to implements a caching functionality.
// An implementation must be safe for concurrent use.
type Cacher interface {
	//// Capacity returns cache capacity.
	//Capacity() int
	//
	//// SetCapacity sets cache capacity.
	//SetCapacity(capacity int)

	// Promote promotes the 'cache node'.
	Promote(n *Node)

	// Ban evicts the 'cache node' and prevent subsequent 'promote'.
	Ban(n *Node)

	// Evict evicts the 'cache node'.
	Evict(n *Node)

	// EvictWithStrategy evicts 'cache node' with a given strategy.
	//
	// It returns the number of the evicted `cache node`
	EvictWithStrategy(st func(blockNum uint64) bool) int

	// EvictAll evicts all 'cache node'.
	EvictAll()

	// Close closes the 'cache tree'
	Close() error
}

// The hash tables implementation is based on:
// "Dynamic-Sized Nonblocking Hash Tables", by Yujie Liu,
// Kunlong Zhang, and Michael Spear.
// ACM Symposium on Principles of Distributed Computing, Jul 2014.

const (
	mInitialSize           = 1 << 4 // initial bucket number
	mOverflowThreshold     = 1 << 5 // threshold number of entry in each bucket
	mOverflowGrowThreshold = 1 << 7 // threshold number of the overflow cache entry

)

type mBucket struct {
	mu     sync.Mutex
	node   []*Node
	frozen bool
}

func (b *mBucket) freeze() []*Node {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.frozen {
		b.frozen = true
	}
	return b.node
}

func (b *mBucket) get(r *Cache, h *mNode, hash uint32, key interface{}, noset bool) (done, added bool, n *Node) {
	b.mu.Lock()

	if b.frozen {
		b.mu.Unlock()
		return
	}

	// Scan the node.
	for _, n := range b.node {
		if n.hash == hash && n.key == key {
			b.mu.Unlock()
			return true, false, n
		}
	}

	// Get only.
	if noset {
		b.mu.Unlock()
		return true, false, nil
	}

	// Create node.
	n = &Node{
		r:    r,
		hash: hash,
		key:  key,
		ref:  1,
	}
	// Add node to bucket.
	b.node = append(b.node, n)
	bLen := len(b.node)
	b.mu.Unlock()

	// Update counter.
	grow := atomic.AddInt32(&r.nodes, 1) >= h.growThreshold
	if bLen > mOverflowThreshold {
		grow = grow || atomic.AddInt32(&h.overflow, 1) >= mOverflowGrowThreshold
	}

	// Grow.
	if grow && atomic.CompareAndSwapInt32(&h.resizeInProgess, 0, 1) {
		nhLen := len(h.buckets) << 1
		nh := &mNode{
			buckets:         make([]unsafe.Pointer, nhLen),
			mask:            uint32(nhLen) - 1,
			pred:            unsafe.Pointer(h),
			growThreshold:   int32(nhLen * mOverflowThreshold),
			shrinkThreshold: int32(nhLen >> 1),
		}
		ok := atomic.CompareAndSwapPointer(&r.mHead, unsafe.Pointer(h), unsafe.Pointer(nh))
		if !ok {
			panic("BUG: failed swapping head")
		}
		go nh.initBuckets()
	}

	return true, true, n
}

func (b *mBucket) delete(r *Cache, h *mNode, hash uint32, key interface{}) (done, deleted bool) {
	b.mu.Lock()

	if b.frozen {
		b.mu.Unlock()
		return
	}

	// Scan the node.
	var (
		n    *Node
		bLen int
	)
	for i := range b.node {
		n = b.node[i]
		if n.key == key {
			if atomic.LoadInt32(&n.ref) == 0 {
				deleted = true
				n.value = nil

				// Remove node from bucket.
				b.node = append(b.node[:i], b.node[i+1:]...)
				bLen = len(b.node)
			}
			break
		}
	}
	b.mu.Unlock()

	if deleted {
		// Update counter.
		atomic.AddInt32(&r.size, int32(n.size)*-1)
		shrink := atomic.AddInt32(&r.nodes, -1) < h.shrinkThreshold
		if bLen >= mOverflowThreshold {
			atomic.AddInt32(&h.overflow, -1)
		}

		// Shrink.
		if shrink && len(h.buckets) > mInitialSize && atomic.CompareAndSwapInt32(&h.resizeInProgess, 0, 1) {
			nhLen := len(h.buckets) >> 1
			nh := &mNode{
				buckets:         make([]unsafe.Pointer, nhLen),
				mask:            uint32(nhLen) - 1,
				pred:            unsafe.Pointer(h),
				growThreshold:   int32(nhLen * mOverflowThreshold),
				shrinkThreshold: int32(nhLen >> 1),
			}
			ok := atomic.CompareAndSwapPointer(&r.mHead, unsafe.Pointer(h), unsafe.Pointer(nh))
			if !ok {
				panic("BUG: failed swapping head")
			}
			go nh.initBuckets()
		}
	}

	return true, deleted
}

type mNode struct {
	buckets         []unsafe.Pointer // []*mBucket
	mask            uint32
	pred            unsafe.Pointer // *mNode
	resizeInProgess int32

	overflow        int32
	growThreshold   int32
	shrinkThreshold int32
}

func (n *mNode) initBucket(i uint32) *mBucket {
	if b := (*mBucket)(atomic.LoadPointer(&n.buckets[i])); b != nil {
		return b
	}

	p := (*mNode)(atomic.LoadPointer(&n.pred))
	if p != nil {
		var node []*Node
		if n.mask > p.mask {
			// Grow.
			pb := (*mBucket)(atomic.LoadPointer(&p.buckets[i&p.mask]))
			if pb == nil {
				pb = p.initBucket(i & p.mask)
			}
			m := pb.freeze()
			// Split nodes.
			for _, x := range m {
				if x.hash&n.mask == i {
					node = append(node, x)
				}
			}
		} else {
			// Shrink.
			pb0 := (*mBucket)(atomic.LoadPointer(&p.buckets[i]))
			if pb0 == nil {
				pb0 = p.initBucket(i)
			}
			pb1 := (*mBucket)(atomic.LoadPointer(&p.buckets[i+uint32(len(n.buckets))]))
			if pb1 == nil {
				pb1 = p.initBucket(i + uint32(len(n.buckets)))
			}
			m0 := pb0.freeze()
			m1 := pb1.freeze()
			// Merge nodes.
			node = make([]*Node, 0, len(m0)+len(m1))
			node = append(node, m0...)
			node = append(node, m1...)
		}
		b := &mBucket{node: node}
		if atomic.CompareAndSwapPointer(&n.buckets[i], nil, unsafe.Pointer(b)) {
			if len(node) > mOverflowThreshold {
				atomic.AddInt32(&n.overflow, int32(len(node)-mOverflowThreshold))
			}
			return b
		}
	}

	return (*mBucket)(atomic.LoadPointer(&n.buckets[i]))
}

func (n *mNode) initBuckets() {
	for i := range n.buckets {
		n.initBucket(uint32(i))
	}
	atomic.StorePointer(&n.pred, nil)
}

// Cache is a 'cache map'.
type Cache struct {
	mu     sync.RWMutex
	mHead  unsafe.Pointer // *mNode
	nodes  int32          // amount of `cache node`
	size   int32          // total size of `cache node`
	cacher Cacher
	closed bool
}

// NewCache creates a new 'cache map'. The cacher is optional and
// may be nil.
func NewCache(cacher Cacher) *Cache {
	h := &mNode{
		buckets:         make([]unsafe.Pointer, mInitialSize),
		mask:            mInitialSize - 1,
		growThreshold:   int32(mInitialSize * mOverflowThreshold),
		shrinkThreshold: 0,
	}
	for i := range h.buckets {
		h.buckets[i] = unsafe.Pointer(&mBucket{})
	}
	r := &Cache{
		mHead:  unsafe.Pointer(h),
		cacher: cacher,
	}
	return r
}

func (r *Cache) getBucket(hash uint32) (*mNode, *mBucket) {
	h := (*mNode)(atomic.LoadPointer(&r.mHead))
	i := hash & h.mask
	b := (*mBucket)(atomic.LoadPointer(&h.buckets[i]))
	if b == nil {
		b = h.initBucket(i)
	}
	return h, b
}

func (r *Cache) delete(n *Node) bool {
	for {
		h, b := r.getBucket(n.hash)
		done, deleted := b.delete(r, h, n.hash, n.key)
		if done {
			return deleted
		}
	}
	return false
}

// Nodes returns number of 'cache node' in the map.
func (r *Cache) Nodes() int {
	return int(atomic.LoadInt32(&r.nodes))
}

// Size returns sums of 'cache node' size in the map.
func (r *Cache) Size() int {
	return int(atomic.LoadInt32(&r.size))
}

// Capacity returns cache capacity.
//func (r *Cache) Capacity() int {
//	if r.cacher == nil {
//		return 0
//	}
//	return r.cacher.Capacity()
//}
//
//// SetCapacity sets cache capacity.
//func (r *Cache) SetCapacity(capacity int) {
//	if r.cacher != nil {
//		r.cacher.SetCapacity(capacity)
//	}
//}

// Get gets the value with the given key.
func (r *Cache) Get(key interface{}) (val interface{}, blockNum uint64) {
	_, val, bnum := r.get(key, nil)
	return val, bnum
}

// Add sets k,v data and the blockNum when inserting into cache.
// It returns true if the data pair is inserted into the cache successfully.
func (r *Cache) Add(key, value interface{}, blockNum uint64) bool {
	added, _, _ := r.get(key, func() (int, interface{}, uint64) {
		return 1, value, blockNum
	})
	return added
}

// Get gets 'cache node' with the given namespace and key.
// If cache node is not found and setFunc is not nil, Get will atomically creates
// the 'cache node' by calling setFunc. Otherwise Get will returns nil.
//
// The returned 'cache handle' should be released after use by calling Release
// method.
func (r *Cache) get(key interface{}, setFunc func() (size int, value interface{}, blockNum uint64)) (added bool, value interface{}, blockNum uint64) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return false, nil, 0
	}

	hash := getHash(key)
	for {
		h, b := r.getBucket(hash)
		done, _, n := b.get(r, h, hash, key, setFunc == nil)
		if done {
			if n != nil {
				n.mu.Lock()
				if setFunc == nil {
					n.mu.Unlock()
					return false, n.value, n.blockNum
				}

				old := n.value
				if old != nil {
					atomic.AddInt32(&r.size, int32(n.size)*-1)
				}
				n.size, n.value, n.blockNum = setFunc()
				if n.value == nil {
					n.size = 0
					n.mu.Unlock()
					n.unref()
					return old != nil, nil, n.blockNum
				}
				atomic.AddInt32(&r.size, int32(n.size))

				n.mu.Unlock()
				if r.cacher != nil {
					r.cacher.Promote(n)
				}
				return true, n.value, n.blockNum
			}

			break
		}
	}
	return false, nil, 0
}

// Delete removes and ban 'cache node' with the given namespace and key.
// A banned 'cache node' will never inserted into the 'cache tree'. Ban
// only attributed to the particular 'cache node', so when a 'cache node'
// is recreated it will not be banned.
//
// Delete return true is such 'cache node' exist.
func (r *Cache) Delete(key interface{}) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return false
	}

	hash := getHash(key)
	for {
		h, b := r.getBucket(hash)
		done, _, n := b.get(r, h, hash, key, true)
		if done {
			if n != nil {
				if r.cacher != nil {
					r.cacher.Ban(n)
				}
				n.unref()
				return true
			}

			break
		}
	}

	return false
}

func (r *Cache) EvictWithStrategy(strategy func(blockNum uint64) bool) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return 0
	}

	if strategy != nil && r.cacher != nil {
		return r.cacher.EvictWithStrategy(strategy)
	}
	return 0
}

// EvictAll evicts all 'cache node'. This will simply call Cacher.EvictAll.
func (r *Cache) Purge() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return
	}

	if r.cacher != nil {
		r.cacher.EvictAll()
	}
}

// Purge purges the 'cache map' and forcefully releases all 'cache node'.
func (r *Cache) Close() error {
	r.mu.Lock()
	if !r.closed {
		r.closed = true

		h := (*mNode)(r.mHead)
		h.initBuckets()

		for i := range h.buckets {
			b := (*mBucket)(h.buckets[i])
			for _, n := range b.node {
				n.value = nil
			}
		}
	}
	r.mu.Unlock()

	// Avoid deadlock.
	if r.cacher != nil {
		if err := r.cacher.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Node is a 'cache node'.
type Node struct {
	r *Cache

	hash     uint32
	blockNum uint64

	mu         sync.Mutex
	size       int
	key, value interface{}

	ref int32

	CacheData unsafe.Pointer
}

// Key returns this 'cache node' key.
func (n *Node) Key() interface{} {
	return n.key
}

// Size returns this 'cache node' size.
func (n *Node) Size() int {
	return n.size
}

// Value returns this 'cache node' value.
func (n *Node) Value() interface{} {
	return n.value
}

// Ref returns this 'cache node' ref counter.
func (n *Node) Ref() int32 {
	return atomic.LoadInt32(&n.ref)
}

// GetHandle returns an handle for this 'cache node'.
func (n *Node) GetHandle() *Handle {
	//if atomic.AddInt32(&n.ref, 1) <= 1 {
	//	panic("BUG: Node.GetHandle on zero ref")
	//}
	return &Handle{unsafe.Pointer(n)}
}

func (n *Node) unref() {
	if atomic.AddInt32(&n.ref, -1) == 0 {
		n.r.delete(n)
	}
}

func (n *Node) unrefLocked() {
	if atomic.AddInt32(&n.ref, -1) == 0 {
		n.r.mu.RLock()
		if !n.r.closed {
			n.r.delete(n)
		}
		n.r.mu.RUnlock()
	}
}

// Handle is a 'cache handle' of a 'cache node'.
type Handle struct {
	n unsafe.Pointer // *Node
}

// Value returns the value of the 'cache node'.
func (h *Handle) Value() interface{} {
	n := (*Node)(atomic.LoadPointer(&h.n))
	if n != nil {
		return n.value
	}
	return nil
}

// Release releases this 'cache handle'.
// It is safe to call release multiple times.
func (h *Handle) Release() {
	nPtr := atomic.LoadPointer(&h.n)
	if nPtr != nil && atomic.CompareAndSwapPointer(&h.n, nPtr, nil) {
		n := (*Node)(nPtr)
		n.unrefLocked()
	}
}

func getHash(key interface{}) uint32 {
	return murmur32(bytes2uint64([]byte(fmt.Sprintf("%s", key))), 0xf00)
}

func murmur32(key uint64, seed uint32) uint32 {
	const (
		m = uint32(0x5bd1e995)
		r = 24
	)

	k1 := uint32(key >> 32)
	k2 := uint32(key)

	k1 *= m
	k1 ^= k1 >> r
	k1 *= m

	k2 *= m
	k2 ^= k2 >> r
	k2 *= m

	h := seed

	h *= m
	h ^= k1
	h *= m
	h ^= k2
	h *= m

	h ^= h >> 13
	h *= m
	h ^= h >> 15

	return h
}

func bytes2uint64(b []byte) uint64 {
	d := sha512.Sum512(b)
	return binary.BigEndian.Uint64(d[:])
}
