package db

import "github.com/syndtr/goleveldb/leveldb/iterator"

type Database interface {
	Put(key []byte, value []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	Close()
	NewBatch() Batch
	NewIterator(prefix []byte) iterator.Iterator
}

type Batch interface {
	Put(key, value []byte) error
	Delete(key []byte) error
	Write() error
	Len() int
}
