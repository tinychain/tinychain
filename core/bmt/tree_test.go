package bmt

import (
	"testing"
	"tinychain/db/leveldb"
	"github.com/stretchr/testify/assert"
	"tinychain/common"
)

var (
	btree = CreateBucketTree()
	root  common.Hash
	db    *leveldb.LDBDatabase
)

func CreateBucketTree() *BucketTree {
	if db == nil {
		db, _ = leveldb.NewLDBDataBase("bucket_tree_test")
	}
	return NewBucketTree(db)
}

func TestBucketTree_WithoutDB(t *testing.T) {
	tree := NewBucketTree(nil)
	writeSet := NewWriteSet()
	writeSet["test1"] = []byte("asdffsdf")
	writeSet["abcd"] = []byte("test2asd")
	writeSet["lslsl"] = []byte("test3f")
	writeSet["werw"] = []byte("test12as")
	writeSet["ffff"] = []byte("FDas")
	writeSet["asdf"] = []byte("asdfff")
	err := tree.Init(nil)
	assert.Nil(t, err)
	err = tree.Prepare(writeSet)
	assert.Nil(t, err)
}

func TestBucketTree_Process(t *testing.T) {
	var batch = db.NewBatch()
	writeSet := NewWriteSet()
	writeSet["test1"] = []byte("asdffsdf")
	writeSet["abcd"] = []byte("test2asd")
	writeSet["lslsl"] = []byte("test3f")
	writeSet["werw"] = []byte("test12as")
	writeSet["ffff"] = []byte("FDas")
	writeSet["asdf"] = []byte("asdfff")
	err := btree.Init(nil)
	assert.Nil(t, err)
	err = btree.Prepare(writeSet)
	assert.Nil(t, err)
	err = btree.Commit(batch)
	assert.Nil(t, err)
	err = batch.Write()
	assert.Nil(t, err)
}

func TestBucketTree_Read(t *testing.T) {
	//for _, bucket := range btree.hashTable.buckets {
	//	if bucket != nil && !bucket.Hash().Nil() {
	//fmt.Printf("%s\n", bucket.Hash().Hex())
	//	}
	//}
	var nilHash common.Hash
	assert.NotEqual(t, nilHash, btree.Hash())
}

func TestBucketTree_Update(t *testing.T) {
	var batch = db.NewBatch()
	oldRoot := btree.Hash()
	newSet := NewWriteSet()
	newSet["lowesyang"] = []byte("lowesyang")
	err := btree.Prepare(newSet)
	assert.Nil(t, err)
	err = btree.Commit(batch)
	assert.Nil(t, err)
	err = batch.Write()
	assert.Nil(t, err)
	assert.NotEqual(t, oldRoot, btree.Hash())
}

func TestBucketTree_LoadFromDB(t *testing.T) {
	newTree := CreateBucketTree()
	root = btree.Hash()
	err := newTree.Init(root.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, root, newTree.Hash())
	assert.Equal(t, btree.llevel, newTree.llevel)
	assert.Equal(t, len(btree.hashTable.BucketHash), len(newTree.hashTable.BucketHash))
}
