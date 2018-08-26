// Copyright (c) 2012, Suryandaru Triandana <syndtr@gmail.com>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"math/rand"
	"testing"
	"time"
)

func BenchmarkLBNCacheAdd(b *testing.B) {
	c := NewCache(NewLBN())

	b.SetParallelism(10)
	b.RunParallel(func(pb *testing.PB) {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		for pb.Next() {
			key := uint64(r.Intn(1000000))
			c.Add(key, key, BLOCK_1)
		}
	})
}

func BenchmarkLBNCacheGet(b *testing.B) {
	c := NewCache(NewLBN())

	b.SetParallelism(10)
	b.RunParallel(func(pb *testing.PB) {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		for pb.Next() {
			key := uint64(r.Intn(1000000))
			c.Get(key)
		}
	})
}
