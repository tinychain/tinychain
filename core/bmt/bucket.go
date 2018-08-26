package bmt

import (
	"tinychain/common"
	"encoding/binary"
	"sync"
	json "github.com/json-iterator/go"
	"bytes"
	"errors"
	"sort"
	"tinychain/db/leveldb"
)

type Bucket struct {
	lock  sync.RWMutex
	H     common.Hash       `json:"hash"`
	Slots map[string][]byte `json:"slots"`
	Keys  []string          `json:"keys"` // store the order of map key
}

func NewBucket() *Bucket {
	return &Bucket{
		Slots: make(map[string][]byte),
	}
}

func (bk *Bucket) Hash() common.Hash {
	return bk.H
}

// Compute hash
func (bk *Bucket) computeHash() common.Hash {
	bk.lock.Lock()
	defer bk.lock.Unlock()
	var bytes []byte
	// Sort the keys array in increasing order
	if !sort.StringsAreSorted(bk.Keys) {
		sort.Strings(bk.Keys)
	}
	for _, key := range bk.Keys {
		bytes = append(bytes, bk.Slots[key]...)
	}
	bk.H = common.Sha256(bytes)
	return bk.H
}

// Check key is existed or not
func (bk *Bucket) hasKey(key string) bool {
	for _, v := range bk.Keys {
		if v == key {
			return true
		}
	}
	return false
}

func (bk *Bucket) addKey(key string) {
	if bk.hasKey(key) {
		return
	}
	bk.lock.Lock()
	defer bk.lock.Unlock()
	bk.Keys = append(bk.Keys, key)
}

func (bk *Bucket) delKey(key string) {
	for i, k := range bk.Keys {
		if k == key {
			bk.Keys = append(bk.Keys[:i], bk.Keys[i+1:]...)
			return
		}
	}
}

func (bk *Bucket) serialize() ([]byte, error) {
	return json.Marshal(bk)
}

func (bk *Bucket) deserialize(d []byte) error {
	return json.Unmarshal(d, bk)
}

// Wrapper of buckets
type HashTable struct {
	db         *BmtDB
	Cap        int           `json:"cap"`
	BucketHash []common.Hash `json:"bucket_hash"`
	buckets    []*Bucket
	dirty      []bool
	lock       sync.RWMutex
}

func NewHashTable(db *BmtDB, cap int) *HashTable {
	return &HashTable{
		db:         db,
		Cap:        cap,
		buckets:    make([]*Bucket, cap, cap),
		BucketHash: make([]common.Hash, cap, cap),
		dirty:      make([]bool, cap, cap),
	}
}

func (ht *HashTable) copy() *HashTable {
	var newBuckets []*Bucket
	newHT := *ht
	for _, bucket := range ht.buckets {
		nb := *bucket
		newBuckets = append(newBuckets, &nb)
	}
	newHT.buckets = newBuckets
	return &newHT
}

func (ht *HashTable) serialize() ([]byte, error) {
	return json.Marshal(ht)
}

func (ht *HashTable) deserialize(d []byte) error {
	return json.Unmarshal(d, ht)
}

func (ht *HashTable) getIndex(key string) uint32 {
	val := binary.BigEndian.Uint32([]byte(key))
	return val % uint32(ht.Cap)
}

func (ht *HashTable) put(key string, value []byte) error {
	var (
		err    error
		bucket *Bucket
	)
	ht.lock.Lock()
	defer ht.lock.Unlock()
	index := ht.getIndex(key)
	bucket = ht.buckets[index]
	if bucket == nil {
		if hash := ht.BucketHash[index]; !hash.Nil() && ht.db != nil {
			bucket, err = ht.db.GetBucket(hash)
			if err != nil {
				return err
			}
		} else {
			bucket = NewBucket()
		}
		ht.buckets[index] = bucket
	}
	oldVal := bucket.Slots[key]
	if bytes.Compare(oldVal, value) != 0 {
		if value == nil {
			bucket.delKey(key)
			delete(bucket.Slots, key)
		} else {
			bucket.addKey(key)
			bucket.Slots[key] = value
		}
		ht.dirty[index] = true
	}
	return nil
}

func (ht *HashTable) get(key string) ([]byte, error) {
	var (
		err    error
		bucket *Bucket
	)
	ht.lock.RLock()
	defer ht.lock.RUnlock()
	index := ht.getIndex(key)
	bucket = ht.buckets[index]
	if bucket == nil {
		if hash := ht.BucketHash[index]; !hash.Nil() && ht.db != nil {
			// Get bucket from db by bucket_hash
			bucket, err = ht.db.GetBucket(ht.BucketHash[index])
			if err != nil {
				return nil, err
			}
			ht.buckets[index] = bucket
		} else {
			return nil, errors.New("value not found")
		}
	}
	return bucket.Slots[key], nil
}

func (ht *HashTable) commit(batch *leveldb.Batch) error {
	if ht.db == nil {
		return ErrDbNotOpen
	}
	ht.lock.Lock()
	defer ht.lock.Unlock()
	for i, dirty := range ht.dirty {
		if dirty {
			bucket := ht.buckets[i]
			err := ht.db.PutBucket(batch, bucket.Hash(), bucket)
			if err != nil {
				return err
			}
		}
		ht.dirty[i] = false
	}
	for i, bucket := range ht.buckets {
		if bucket != nil {
			ht.BucketHash[i] = bucket.Hash()
		}
	}
	return nil
}

func (ht *HashTable) purge() {
	ht.buckets = make([]*Bucket, ht.Cap, ht.Cap)
}
