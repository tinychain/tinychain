package state

import (
	"tinychain/db/leveldb"
	"github.com/hashicorp/golang-lru"
	"tinychain/common"
)

const (
	cacheSize       = 128
	KeyContractCode = "c"
)

// CacheDB is used to store contract code
// "c" + contract_code_hash => code

type cacheDB struct {
	db        *leveldb.LDBDatabase
	codeCache *lru.Cache
}

func newCacheDB(db *leveldb.LDBDatabase) *cacheDB {
	l, _ := lru.New(cacheSize)
	return &cacheDB{
		db:        db,
		codeCache: l,
	}
}

func (db *cacheDB) GetCode(codeHash common.Hash) ([]byte, error) {
	if code, ok := db.codeCache.Get(codeHash.Bytes()); ok {
		return code.([]byte), nil
	}
	key := append([]byte(KeyContractCode), codeHash.Bytes()...)
	code, err := db.db.Get(key)
	if err != nil {
		return nil, err
	}
	db.codeCache.Add(codeHash, code)
	return code, nil
}

func (db *cacheDB) PutCode(codeHash common.Hash, code []byte) error {
	key := append([]byte(KeyContractCode), codeHash.Bytes()...)
	err := db.db.Put(key, code)
	if err != nil {
		log.Errorf("Failed to put code with hash %s, %s", codeHash, err)
		return err
	}
	return nil
}

