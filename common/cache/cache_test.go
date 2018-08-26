// Copyright (c) 2012, Suryandaru Triandana <syndtr@gmail.com>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	BLOCK_1 uint64 = iota
	BLOCK_2
	BLOCK_3
	BLOCK_4
)

type int32o int32

func set(c *Cache, key, value interface{}, blockNum uint64) bool {
	return c.Add(key, value, blockNum)
}

type cacheMapTestParams struct {
	nobjects, concurrent, repeat int
}

func TestCacheMap(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var params []cacheMapTestParams
	if testing.Short() {
		params = []cacheMapTestParams{
			{1000, 20, 3},
			{10000, 50, 10},
		}
	} else {
		params = []cacheMapTestParams{
			{10000, 50, 3},
			{100000, 100, 10},
		}
	}

	var (
		objects [][]int32o
	)

	for _, x := range params {
		objects = append(objects, make([]int32o, x.nobjects))
	}

	c := NewCache(nil)

	wg := new(sync.WaitGroup)
	var done int32

	for ns, x := range params {
		for i := 0; i < x.concurrent; i++ {
			wg.Add(1)
			go func(i, repeat int, objects []int32o) {
				defer wg.Done()
				r := rand.New(rand.NewSource(time.Now().UnixNano()))

				for j := len(objects) * repeat; j >= 0; j-- {
					key := r.Intn(len(objects))
					o := int32(objects[key])
					atomic.AddInt32(&o, 1)
					added := c.Add(key, objects[key], BLOCK_1)
					//if v := h.Value().(*int32o); v != &objects[key] {
					//	t.Fatalf("#%d invalid value: want=%p got=%p", ns, &objects[key], v)
					//}
					if !added {
						t.Fatalf("%v:%v cannot be added to LBN cache", key, objects[key])
					}

				}
			}(i, x.repeat, objects[ns])
		}
	}

	go func() {
		values := make([]interface{}, 100000)
		for atomic.LoadInt32(&done) == 0 {
			for i := range values {
				c.Add(i, 1, BLOCK_1)
				values[i] = 1
			}
		}
	}()

	wg.Wait()

	atomic.StoreInt32(&done, 1)

	//for ns, objects0 := range objects {
	//	for i, o := range objects0 {
	//		if o == 0 {
	//			t.Fatalf("invalid object #%d.%d: ref=%d", ns, i, o)
	//		}
	//	}
	//}
}

func TestCacheMap_NodesAndSize(t *testing.T) {
	c := NewCache(nil)
	if c.Nodes() != 0 {
		t.Errorf("invalid nodes counter: want=%d got=%d", 0, c.Nodes())
	}
	if c.Size() != 0 {
		t.Errorf("invalid size counter: want=%d got=%d", 0, c.Size())
	}
	set(c, 0, 0, BLOCK_1)
	set(c, 1, 1, BLOCK_2)
	set(c, 2, 2, BLOCK_2)
	for _, key := range []interface{}{0, 1, 2} {
		val, _ := c.Get(key)
		if val == nil {
			t.Errorf("miss for key %d", key)
		}

		if val.(int) != key.(int) {
			t.Errorf("val %v is not equal to key %v", val, key)
		}
	}

	if c.Nodes() != 3 {
		t.Errorf("invalid nodes counter: want=%d got=%d", 3, c.Nodes())
	}
	if c.Size() != 3 {
		t.Errorf("invalid size counter: want=%d got=%d", 3, c.Size())
	}
}

func TestCacheMap_SetValueWithSameKey(t *testing.T) {
	c := NewCache(nil)
	set(c, 0, 1, BLOCK_1)
	val, blockNum := c.Get(0)
	if val == nil {
		t.Errorf("Cache.get return nil")
	}
	if val != 1 {
		t.Errorf("Cache.get on key 0 want=%d, got=%d", 1, val)
	}
	if blockNum != BLOCK_1 {
		t.Errorf("Cache.get on key 0 block number want=%d,got=%d", BLOCK_1, blockNum)
	}
	if c.Nodes() != 1 {
		t.Errorf("invalid nodes counter: want=%d got=%d", 1, c.Nodes())
	}
	if c.Size() != 1 {
		t.Errorf("invalid size counter: want=%d got=%d", 1, c.Size())
	}

	set(c, 0, 2, BLOCK_1)
	val, _ = c.Get(0)
	if val == nil {
		t.Errorf("Cache.get return nil")
	}
	if val != 2 {
		t.Errorf("Cache.get on key 0 want=%d, got=%d", 2, val)
	}
	if c.Nodes() != 1 {
		t.Errorf("invalid nodes counter: want=%d got=%d", 1, c.Nodes())
	}
	if c.Size() != 1 {
		t.Errorf("invalid size counter: want=%d got=%d", 1, c.Size())
	}

	set(c, 1, 3, BLOCK_2)
	val, _ = c.Get(1)
	if val != 3 {
		t.Errorf("Cache.get on key 0 want=%d, got=%d", 3, val)
	}
	if c.Nodes() != 2 {
		t.Errorf("invalid nodes counter: want=%d got=%d", 2, c.Nodes())
	}
	if c.Size() != 2 {
		t.Errorf("invalid size counter: want=%d got=%d", 2, c.Size())
	}
}

//func TestLBNCache_Capacity(t *testing.T) {
//	c := NewCache(NewLBN())
//	if c.Capacity() != 10 {
//		t.Errorf("invalid capacity: want=%d got=%d", 10, c.Capacity())
//	}
//	set(c, 0, 1, BLOCK_1)
//	set(c, 0, 2, BLOCK_1)
//	set(c, 1, 1, BLOCK_2)
//	set(c, 2, 1, BLOCK_2)
//	set(c, 3, 2, BLOCK_3)
//	set(c, 4, 3, BLOCK_3)
//	set(c, 5, 4, BLOCK_4)
//	set(c, 6, 5, BLOCK_4)
//	if c.Nodes() != 7 {
//		t.Errorf("invalid nodes counter: want=%d got=%d", 7, c.Nodes())
//	}
//	if c.Size() != 7 {
//		t.Errorf("invalid size counter: want=%d got=%d", 7, c.Size())
//	}
//}

func TestCacheMap_NilValue(t *testing.T) {
	c := NewCache(NewLBN())
	added := c.Add(0, nil, BLOCK_1)
	if added {
		t.Error("the result of inserting operation should not be true")
	}
	if c.Nodes() != 0 {
		t.Errorf("invalid nodes counter: want=%d got=%d", 0, c.Nodes())
	}
	if c.Size() != 0 {
		t.Errorf("invalid size counter: want=%d got=%d", 0, c.Size())
	}
}

func TestCacheMap_UpdateBlockNum(t *testing.T) {
	c := NewCache(NewLBN())
	c.Add(0, 1, BLOCK_1)
	added := c.Add(0, 1, BLOCK_2)
	if !added {
		t.Error("the result of inserting operation should not be false")
	}

	_, num := c.Get(0)
	if num != BLOCK_2 {
		t.Error("invalid block num retrieved: want=%d got=%d", BLOCK_2, num)
	}
}

func TestLBNCache_GetLatency(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	const (
		concurrentSet = 30
		concurrentGet = 3
		duration      = 3 * time.Second
		delay         = 3 * time.Millisecond
		maxkey        = 100000
	)

	var (
		set, getHit, getAll        int32
		getMaxLatency, getDuration int64
	)

	c := NewCache(NewLBN())
	wg := &sync.WaitGroup{}
	until := time.Now().Add(duration)
	for i := 0; i < concurrentSet; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for time.Now().Before(until) {
				time.Sleep(delay)
				c.Add(r.Intn(maxkey), 1, BLOCK_1)
				atomic.AddInt32(&set, 1)
			}
		}(i)
	}
	for i := 0; i < concurrentGet; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for {
				mark := time.Now()
				if mark.Before(until) {
					val, _ := c.Get(r.Intn(maxkey))
					latency := int64(time.Now().Sub(mark))
					m := atomic.LoadInt64(&getMaxLatency)
					if latency > m {
						atomic.CompareAndSwapInt64(&getMaxLatency, m, latency)
					}
					atomic.AddInt64(&getDuration, latency)
					if val != nil {
						atomic.AddInt32(&getHit, 1)
					}
					atomic.AddInt32(&getAll, 1)
				} else {
					break
				}
			}
		}(i)
	}

	wg.Wait()
	getAvglatency := time.Duration(getDuration) / time.Duration(getAll)
	t.Logf("set=%d getHit=%d getAll=%d getMaxLatency=%v getAvgLatency=%v",
		set, getHit, getAll, time.Duration(getMaxLatency), getAvglatency)

	if getAvglatency > delay/3 {
		t.Errorf("get avg latency > %v: got=%v", delay/3, getAvglatency)
	}
}

func TestLBNCache_HitMiss(t *testing.T) {
	cases := []struct {
		key   uint64
		value string
	}{
		{1, "vvvvvvvvv"},
		{100, "v1"},
		{0, "v2"},
		{12346, "v3"},
		{777, "v4"},
		{999, "v5"},
		{7654, "v6"},
		{2, "v7"},
		{3, "v8"},
		{9, "v9"},
	}

	c := NewCache(NewLBN())
	for i, x := range cases {
		set(c, x.key, x.value, BLOCK_1)
		for j, y := range cases {
			val, blockNum := c.Get(y.key)
			if j <= i {
				// should hit
				if val == nil {
					t.Errorf("case '%d' iteration '%d' is miss", i, j)
				} else {
					if blockNum != BLOCK_1 {
						t.Errorf("case '%d' iteration '%d' block num '%d' is not match '%d' ", i, j, blockNum, BLOCK_1)
					}
					if x := val.(string); x != y.value {
						t.Errorf("case '%d' iteration '%d' has invalid value got '%s', want '%s'", i, j, x, y.value)
					}
				}
			} else {
				// should miss
				if val != nil {
					t.Errorf("case '%d' iteration '%d' is hit , value '%s'", i, j, val.(string))
				}
			}
		}
	}

	for i, x := range cases {
		deleted := c.Delete(x.key)
		if !deleted {
			t.Errorf("case '%d' interation key '%d' is not deleted", i, x.key)
		}

		for j, y := range cases {
			val, _ := c.Get(y.key)
			if j > i {
				// should hit
				if val == nil {
					t.Errorf("case '%d' iteration '%d' is miss", i, j)
				} else {
					if x := val.(string); x != y.value {
						t.Errorf("case '%d' iteration '%d' has invalid value got '%s', want '%s'", i, j, x, y.value)
					}
				}
			} else {
				// should miss
				if val != nil {
					t.Errorf("case '%d' iteration '%d' is hit, value '%s'", i, j, val.(string))
				}
			}
		}
	}
}

func getStrategy(bnum uint64) func(blockNum uint64) bool {
	return func(blockNum uint64) bool {
		return blockNum <= bnum
	}
}

func TestLBNCache_Eviction(t *testing.T) {

	c := NewCache(NewLBN())
	set(c, 1, 1, BLOCK_1)
	set(c, 2, 2, BLOCK_1)
	set(c, 3, 3, BLOCK_2)
	set(c, 4, 4, BLOCK_2)
	set(c, 5, 5, BLOCK_1)
	set(c, 6, 6, BLOCK_3)
	set(c, 7, 7, BLOCK_3)
	set(c, 8, 8, BLOCK_4)
	set(c, 9, 9, BLOCK_1)

	for _, key := range []interface{}{1, 2, 5, 9} {
		val, blockNum := c.Get(key)
		if val == nil {
			t.Errorf("miss for key '%d'", key)
		} else {
			if blockNum != BLOCK_1 {
				t.Errorf("invalid blockNum for '%d' want '%d',got '%d'", key, BLOCK_1, blockNum)
			}
			if x := val; x != key {
				t.Errorf("invalid value for key '%d' want '%d', got '%d'", key, key, x)
			}
		}
	}

	c.EvictWithStrategy(getStrategy(BLOCK_1))

	for _, key := range []interface{}{9, 2, 5, 1} {
		val, _ := c.Get(key)
		if val != nil {
			t.Errorf("hit for key '%d'", key)
		}
	}

	for _, key := range []interface{}{3, 4} {
		val, blockNum := c.Get(key)
		if val == nil {
			t.Errorf("miss for key '%d'", key)
		} else {
			if blockNum != BLOCK_2 {
				t.Errorf("invalid blockNum for '%d' want '%d',got '%d'", key, BLOCK_1, blockNum)
			}
			if x := val; x != key {
				t.Errorf("invalid value for key '%d' want '%d', got '%d'", key, key, x)
			}
		}
	}

	c.EvictWithStrategy(getStrategy(BLOCK_2))

	for _, key := range []interface{}{3, 4} {
		val, _ := c.Get(key)
		if val != nil {
			t.Errorf("hit for key '%d'", key)
		}
	}

	for _, key := range []interface{}{6, 7, 8} {
		val, _ := c.Get(key)
		if val == nil {
			t.Errorf("miss for key '%d'", key)
		} else {
			if x := val; x != key {
				t.Errorf("invalid value for key '%d' want '%d', got '%d'", key, key, x)
			}
		}
	}

	c.EvictWithStrategy(getStrategy(BLOCK_4))

	for _, key := range []interface{}{6, 7, 8} {
		val, _ := c.Get(key)
		if val != nil {
			t.Errorf("hit for key '%d'", key)
		}
	}
}

func TestLBNCache_Evict(t *testing.T) {
	c := NewCache(NewLBN())
	set(c, 1, 1, BLOCK_1)
	set(c, 2, 2, BLOCK_1)
	set(c, 3, 4, BLOCK_1)
	set(c, 4, 5, BLOCK_1)
	set(c, 5, 6, BLOCK_1)
	set(c, 6, 7, BLOCK_1)

	for key := 1; key <= 6; key++ {
		for key := 1; key < 3; key++ {
			val, blockNum := c.Get(key)
			if blockNum != BLOCK_1 {
				t.Errorf("Cache.Get on key %d blockNum want %d, got %d", key, BLOCK_1, blockNum)
			}
			if val == nil {
				t.Errorf("Cache.Get on key %d return nil", key)
			}
		}
	}

	if ok := c.Delete(1); !ok {
		t.Error("first Cache.Evict on key 1 return false")
	}
	if ok := c.Delete(1); ok {
		t.Error("second Cache.Evict on key 1 return true")
	}
	if val, _ := c.Get(1); val != nil {
		t.Errorf("Cache.Get on key 1 return non-nil: %v", val)
	}

	c.Purge()
	for key := 1; key <= 6; key++ {
		if val, _ := c.Get(key); val != nil {
			t.Errorf("Cache.Get on key %d return non-nil: %v", key, val)
		}
	}
}

func TestLBNCache_Purge(t *testing.T) {
	c := NewCache(NewLBN())
	set(c, 1, 1, BLOCK_1)
	set(c, 2, 2, BLOCK_1)
	set(c, 3, 4, BLOCK_1)
	set(c, 4, 5, BLOCK_1)
	set(c, 5, 6, BLOCK_1)
	set(c, 6, 7, BLOCK_1)

	c.Purge()

	for _, key := range []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9} {
		if val, _ := c.Get(key); val != nil {
			t.Errorf("Cache.Get on key %v return non-nil: %v", key, val)
		}
	}
}

func TestLBNCache_Delete(t *testing.T) {
	c := NewCache(NewLBN())
	set(c, 1, 1, BLOCK_1)
	set(c, 2, 2, BLOCK_1)

	if ok := c.Delete(1); !ok {
		t.Error("Cache.Delete on #1 return false")
	}
	if val, _ := c.Get(1); val != nil {
		t.Errorf("Cache.Get on #1 return non-nil: %v", val)
	}
	if ok := c.Delete(1); ok {
		t.Error("Cache.Delete on #1 return true")
	}

	val, _ := c.Get(2)
	if val == nil {
		t.Error("Cache.Get on #2 return nil")
	}
	if ok := c.Delete(2); !ok {
		t.Error("(1) Cache.Delete on #2 return false")
	}
	if ok := c.Delete(2); ok {
		t.Error("(2) Cache.Delete on #2 return true")
	}

	set(c, 3, 3, BLOCK_1)
	set(c, 4, 4, BLOCK_1)

	for key := 3; key <= 4; key++ {
		val, _ := c.Get(key)
		if val == nil {
			t.Errorf("Cache.Get on #%d return nil", key)
		}
	}

	if val, _ := c.Get(2); val != nil {
		t.Errorf("Cache.Get on #2 return non-nil: %v", val)
	}
}

func TestLBNCache_Close(t *testing.T) {

	c := NewCache(NewLBN())
	set(c, 1, 1, BLOCK_1)
	set(c, 2, 2, BLOCK_1)

	added := set(c, 3, 3, BLOCK_1)
	if !added {
		t.Error("Cache.Get on key 3 return false")
	}
	if ok := c.Delete(3); !ok {
		t.Error("Cache.Delete on key 3 return false")
	}

	c.Close()

	if val, _ := c.Get(1); val != nil {
		t.Errorf("Cache.Get on key 1 return non-nil: %v", val)
	}

	if val, _ := c.Get(2); val != nil {
		t.Errorf("Cache.Get on key 2 return non-nil: %v", val)
	}

	if val, _ := c.Get(3); val != nil {
		t.Errorf("Cache.Get on key 3 return non-nil: %v", val)
	}

	added = set(c, 1, 1, BLOCK_1)
	if added {
		t.Error("Cache.Get on key 1 return true")
	}

	if val, _ := c.Get(1); val != nil {
		t.Errorf("Cache.Get on key 1 return non-nil: %v", val)
	}

}

func TestLBNCache_Resize(t *testing.T) {
	var i int32
	c := NewCache(NewLBN())
	head := (*mNode)(atomic.LoadPointer(&c.mHead))
	initialMax := head.growThreshold
	initialMin := head.shrinkThreshold
	for i = 0; i < initialMax+3; i++ {
		if i < int32(len(head.buckets)/2) {
			c.Add(i, i, BLOCK_1)
		} else {
			c.Add(i, i, BLOCK_2)
		}
	}

	head = (*mNode)(atomic.LoadPointer(&c.mHead))
	if initialMax == head.growThreshold || initialMin == head.shrinkThreshold {
		t.Errorf("Cache cannot resize")
	}

	currMax := head.growThreshold
	currMin := head.shrinkThreshold

	c.EvictWithStrategy(func(blockNum uint64) bool {
		return blockNum == BLOCK_2
	})

	head = (*mNode)(atomic.LoadPointer(&c.mHead))
	if currMax == head.growThreshold || currMin == head.shrinkThreshold {
		t.Errorf("Cache cannot resize")
	}

}
