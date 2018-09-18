package db

import (
	"bytes"
	"fmt"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var log *logging.Logger // package-level logger
const (
	LEVEL_DB_PATH = "dbConfig.leveldbPath"
)

func init() {
	log = logging.MustGetLogger("hyperdb/leveldb_lru")
}

// the Database for LevelDB
// LDBDatabase implements the DataBase interface
type LDBDatabase struct {
	path string
	db   *leveldb.DB
}

// NewLDBDataBase new a LDBDatabase instance
// require a data filepath
// return *LDBDataBase and  will
// return an error with type of
// ErrCorrupted if corruption detected in the DB. Corrupted
// DB can be recovered with Recover function.
// the return *LDBDatabase is goruntine-safe
// the LDBDataBase instance must be close after use, by calling Close method
func NewLDBDataBase(filepath string) (*LDBDatabase, error) {
	db, err := leveldb.OpenFile(filepath, nil)
	return &LDBDatabase{
		path: filepath,
		db:   db,
	}, err
}

// Put sets value for the given key, if the key exists, it will overwrite
// the value
func (self *LDBDatabase) Put(key []byte, value []byte) error {
	return self.db.Put(key, value, nil)
}

// Get gets value for the given key, it returns ErrNotFound if
// the Database does not contains the key
func (self *LDBDatabase) Get(key []byte) ([]byte, error) {
	dat, err := self.db.Get(key, nil)
	if err != nil {
		err = errors.New(fmt.Sprintf("%s not found", string(key)))
	}
	return dat, err
}

// Delete deletes the value for the given key
func (self *LDBDatabase) Delete(key []byte) error {
	return self.db.Delete(key, nil)
}

// NewIterator returns a Iterator for traversing the database
func (self *LDBDatabase) NewIterator(prefix []byte) iterator.Iterator {
	return self.db.NewIterator(util.BytesPrefix(prefix), nil)
}

func (self *LDBDatabase) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
	return self.db.NewIterator(util.BytesPrefix(prefix), nil)
}

//Destroy, clean the whole database,
//warning: bad performance if to many data in the db
func (self *LDBDatabase) Destroy() error {
	return self.DestroyByRange(nil, nil)
}

//DestroyByRange, clean data which key in range [start, end)
func (self *LDBDatabase) DestroyByRange(start, end []byte) error {
	if bytes.Compare(start, end) > 0 {
		return errors.Errorf("start key: %v, is bigger than end key: %v", start, end)
	}
	it := self.db.NewIterator(&util.Range{Start: start, Limit: end}, nil)
	for it.Next() {
		err := self.Delete(it.Key())
		if err != nil {
			return err
		}
	}
	return nil
}

// Close close the LDBDataBase
func (self *LDBDatabase) Close() {
	self.db.Close()
}

// LDB returns *leveldb.DB instance
func (self *LDBDatabase) LDB() *leveldb.DB {
	return self.db
}

// NewBatch returns a Batch instance
// it allows batch-operation
func (db *LDBDatabase) NewBatch() Batch {
	return &ldbBatch{db: db.db, b: new(leveldb.Batch)}
}

// The Batch for LevelDB
// ldbBatch implements the Batch interface
type ldbBatch struct {
	db *leveldb.DB
	b  *leveldb.Batch
}

// Put put the key-value to ldbBatch
func (b *ldbBatch) Put(key, value []byte) error {
	b.b.Put(key, value)
	return nil
}

// Delete delete the key-value to ldbBatch
func (b *ldbBatch) Delete(key []byte) error {
	b.b.Delete(key)
	return nil
}

// Write write batch-operation to database
func (b *ldbBatch) Write() error {
	return b.db.Write(b.b, nil)
}

func (b *ldbBatch) Len() int {
	return b.b.Len()
}
