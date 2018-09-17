// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package event

import (
	"errors"
	"reflect"
	"sync"
)

var (
	errBadChannel = errors.New("event: Subscribe argument does not have sendable channel type")
	errDuplicate  = errors.New("event: duplicate in Subscribe")
)

// This is the index of the first actual subscription channel in sendCases.
// sendCases[0] is a SelectRecv case for the removeSub channel.
const firstSubSendCase = 1

// feed implements one-to-many subscriptions where the carrier of events is a channel.
// Values sent to a feed are delivered to all subscribed channels simultaneously.
//
// feeds can only be used with a single type. The type is determined by the first Send or
// Subscribe operation. Subsequent calls to these methods panic if the type does not
// match.
type feed struct {
	subs      []*muxSub
	sendCases caseList
	sendLock  chan struct{} //
	removeSub chan *muxSub

	// The msgBox holds newly subscribed channels until they are added to sendCases
	mu     sync.Mutex
	msgBox caseList
	etype  reflect.Type
	closed bool
}

func newFeed() *feed {
	rmSub := make(chan *muxSub)
	feed := &feed{
		sendLock:  make(chan struct{}, 1),
		removeSub: rmSub,
		sendCases: []reflect.SelectCase{{Chan: reflect.ValueOf(rmSub), Dir: reflect.SelectRecv}},
	}
	feed.sendLock <- struct{}{}
	return feed
}

// Add adds a channel to the feed. Future sends will be delivered on the channel
// until the subscription is canceled. All channels added must have the same element type.
//
// The channel should have ample buffer space to avoid blocking other subscribers.
// Slow subscribers are not dropped.
//
// Modify func subscribe of Ethereum, and not return subscription
func (f *feed) add(sub *muxSub) error {
	channel := sub.ch()
	chanType := reflect.TypeOf(channel)
	chanVal := reflect.ValueOf(channel)
	if chanType.ChanDir()&reflect.SendDir == 0 {
		panic(errBadChannel)
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	// check sub is existed or not
	for _, ms := range f.subs {
		if ms == sub {
			return errDuplicate
		}
	}

	cas := reflect.SelectCase{Dir: reflect.SelectSend, Chan: chanVal}
	f.msgBox = append(f.msgBox, cas)
	f.subs = append(f.subs, sub)
	return nil
}

// remove delete subscription from subs
// it treats the subscription existing in the subs list acquiescently
func (f *feed) remove(ms *muxSub) {
	channel := ms.ch()
	f.mu.Lock()
	f.del(ms)
	index := f.msgBox.find(channel)
	if index != -1 {
		f.msgBox = f.msgBox.del(index)
		f.mu.Unlock()
		return
	}
	f.mu.Unlock()

	select {
	case f.removeSub <- ms:
	case <-f.sendLock:
		// No send in progress, delete the channel from the sendCases
		f.sendCases = f.sendCases.del(f.sendCases.find(channel))
		f.sendLock <- struct{}{}
	}
}

func (f *feed) send(value interface{}) int {
	var nsent int
	rvalue := reflect.ValueOf(value)

	// Get the lock
	<-f.sendLock

	f.mu.Lock()
	if f.closed {
		return 0
	}
	f.sendCases = append(f.sendCases, f.msgBox...)
	f.msgBox = nil
	f.mu.Unlock()

	for i := firstSubSendCase; i < len(f.sendCases); i++ {
		f.sendCases[i].Send = rvalue
	}

	cases := f.sendCases
	for len(cases) > 0 {
		// Fast path: try sending without blocking before adding to the select set.
		// This should usually succeed if subscribers are fast enough and have free
		// buffer space.
		for i := firstSubSendCase; i < len(cases); i++ {
			if cases[i].Chan.TrySend(rvalue) {
				nsent++
				cases = cases.deactivate(i)
				i--
			}
		}

		if len(cases) == firstSubSendCase {
			break
		}
		// Select on all the receivers, waiting for them to unblock.
		chosen, recv, _ := reflect.Select(cases)
		if chosen == 0 /* <-f.removeSub */ {
			ms := recv.Interface().(*muxSub)
			index := f.sendCases.find(ms.ch())
			f.sendCases = f.sendCases.del(index)
			if index >= 0 && index < len(cases) {
				cases = f.sendCases[:len(cases)-1]
			}
		} else {
			cases = cases.deactivate(chosen)
			nsent++
		}
	}

	for i := firstSubSendCase; i < len(f.sendCases); i++ {
		f.sendCases[i].Send = reflect.Value{}
	}
	f.sendLock <- struct{}{}
	return nsent
}

func (f *feed) find(ms *muxSub) int {
	for i, sub := range f.subs {
		if sub == ms {
			return i
		}
	}
	return -1
}

// The caller should hold the lock
func (f *feed) del(ms *muxSub) {
	index := f.find(ms)
	if index == -1 {
		return
	}
	f.subs = append(f.subs[:index], f.subs[index+1:]...)
}

// close the feed, and can no longer be used to transfer value
func (f *feed) close() {
	if f.closed {
		return
	}
	<-f.sendLock

	f.mu.Lock()
	defer f.mu.Unlock()
	close(f.removeSub)
	f.sendCases = nil
	for _, sub := range f.subs {
		sub.close()
	}
	f.closed = true
	close(f.sendLock)
}

type caseList []reflect.SelectCase

func (c caseList) find(channel interface{}) int {
	for i, cas := range c {
		if cas.Chan.Interface() == channel {
			return i
		}
	}
	return -1
}

func (c caseList) del(i int) caseList {
	return append(c[:i], c[i+1:]...)
}

func (c caseList) deactivate(i int) caseList {
	last := len(c) - 1
	c[i], c[last] = c[last], c[i]
	return c[:last]
}
