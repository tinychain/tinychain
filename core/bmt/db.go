package bmt

import (
	"github.com/tinychain/tinychain/common"
	tdb "github.com/tinychain/tinychain/db"
)

const (
	NodeKeyPrefix      = "n" // "n" + node hash
	HashTableKeyPrefix = "t" // "t" + root node hash
	BucketKeyPrefix    = "s" // "s" + slot hash
)

type BmtDB struct {
	db tdb.Database
}

func NewBmtDB(db tdb.Database) *BmtDB {
	return &BmtDB{
		db: db,
	}
}

func (bdb *BmtDB) GetNode(key common.Hash) (*MerkleNode, error) {
	data, err := bdb.db.Get([]byte(NodeKeyPrefix + key.String()))
	if err != nil {
		return nil, err
	}
	node := &MerkleNode{db: bdb}
	node.deserialize(data)
	node.childNodes = make([]*MerkleNode, len(node.Children))
	return node, nil
}

func (bdb *BmtDB) PutNode(batch tdb.Batch, key common.Hash, node *MerkleNode) error {
	data, err := node.serialize()
	if err != nil {
		return err
	}
	batch.Put([]byte(NodeKeyPrefix+key.String()), data)
	return nil
}

func (bdb *BmtDB) GetBucket(key common.Hash) (*Bucket, error) {
	data, err := bdb.db.Get([]byte(BucketKeyPrefix + key.String()))
	if err != nil {
		return nil, err
	}
	bucket := &Bucket{}
	err = bucket.deserialize(data)
	if err != nil {
		return nil, err
	}
	return bucket, nil
}

func (bdb *BmtDB) PutBucket(batch tdb.Batch, key common.Hash, bucket *Bucket) error {
	data, err := bucket.serialize()
	if err != nil {
		return nil
	}
	batch.Put([]byte(BucketKeyPrefix+key.String()), data)
	return nil
}

func (bdb *BmtDB) GetHashTable(key common.Hash) (*HashTable, error) {
	data, err := bdb.db.Get([]byte(HashTableKeyPrefix + key.String()))
	if err != nil {
		return nil, err
	}
	ht := &HashTable{}
	err = ht.deserialize(data)
	if err != nil {
		return nil, err
	}
	return ht, nil
}

func (bdb *BmtDB) PutHashTable(batch tdb.Batch, key common.Hash, ht *HashTable) error {
	data, err := ht.serialize()
	if err != nil {
		return err
	}
	batch.Put([]byte(HashTableKeyPrefix+key.String()), data)
	return nil
}
