package event

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// ErrMuxClosed is returned when Posting on a closed TypeMux.
var (
	typeMux      *TypeMux
	ErrMuxClosed = errors.New("event: mux closed")
)

// A TypeMux dispatches events to registered receivers. Receivers can be
// registered to handle events of certain type. Any operation
// called after mux is stopped will return ErrMuxClosed.
type TypeMux struct {
	mu      sync.RWMutex
	feeds   map[reflect.Type]*feed
	stopped bool
}

func newTypeMux() *TypeMux {
	return &TypeMux{
		feeds: make(map[reflect.Type]*feed),
	}
}

func (mux *TypeMux) Subscribe(types ...interface{}) Subscription {
	sub := newMuxSub(mux)
	mux.mu.Lock()
	defer mux.mu.Unlock()

	if mux.stopped {
		sub.close()
	} else {
		var feed *feed
		for _, typ := range types {
			rtyp := reflect.TypeOf(typ)
			if feed = mux.feeds[rtyp]; feed == nil {
				feed = newFeed()
				mux.feeds[rtyp] = feed
			}
			if err := feed.add(sub); err == errDuplicate {
				panic(errors.New(fmt.Sprintf("event: duplicate type %s in Subscribe", rtyp)))
			}
		}
	}

	return sub
}

func (mux *TypeMux) del(ms *muxSub) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()
	for _, feed := range mux.feeds {
		if pos := feed.find(ms); pos != -1 {
			feed.remove(ms)
		}
	}
}

// Stop closed the mux. The mux can no longer be used.
func (mux *TypeMux) Stop() {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	if mux.stopped {
		return
	}
	for _, feed := range mux.feeds {
		feed.close()
	}
	mux.stopped = true
}

// Post send ev to each feed of the same type with ev.
// The caller is better to call this func in a new goroutine
func (mux *TypeMux) Post(ev interface{}) error {
	mux.mu.RLock()
	if mux.stopped {
		return ErrMuxClosed
	}
	rtyp := reflect.TypeOf(ev)
	feed := mux.feeds[rtyp]
	mux.mu.RUnlock()

	if feed != nil {
		feed.send(ev)
	}
	return nil
}

func GetEventhub() *TypeMux {
	if typeMux == nil {
		typeMux = newTypeMux()
	}
	return typeMux
}
