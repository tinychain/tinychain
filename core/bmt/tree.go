// Bucket merkle tree implementaion
package bmt

import (
	"bytes"
	"errors"
	"sync"
	"tinychain/common"
	tdb "tinychain/db"
)

var (
	log          = common.GetLogger("bucket_tree")
	ErrDbNotOpen = errors.New("db not open")
)

const (
	defaultHashTableCap = 100000
	defaultAggreation   = 5
)

// Write set for tree prepare
type WriteSet map[string][]byte

func NewWriteSet() WriteSet {
	return make(WriteSet)
}

type BucketTree struct {
	db         *BmtDB
	Capacity   int
	Aggreation int
	llevel     int        // the loweset level of tree
	node       sync.Map   // map[Position]*MerkleNode
	hashTable  *HashTable // dirty data hash table
	dirty      bool
}

func NewBucketTree(db tdb.Database) *BucketTree {
	// v1.0
	return &BucketTree{
		Capacity:   defaultHashTableCap,
		Aggreation: defaultAggreation,
		db:         NewBmtDB(db),
	}
}

func (bt *BucketTree) Hash() common.Hash {
	node, _ := bt.getNode(newPos(0, 0))
	return node.Hash()
}

func (bt *BucketTree) getNode(pos *Position) (*MerkleNode, error) {
	node, ok := bt.node.Load(*pos)
	if !ok {
		return nil, errors.New("node not found")
	}
	return node.(*MerkleNode), nil
}

func (bt *BucketTree) putNode(pos *Position, node *MerkleNode) {
	bt.node.Store(*pos, node)
}

func (bt *BucketTree) LowestLevel() int {
	return bt.llevel
}

// Init constructing the tree structure
func (bt *BucketTree) Init(rootHash []byte) error {
	var (
		root *MerkleNode
		err  error
	)
	if rootHash == nil || bytes.Compare(rootHash, []byte{}) == 0 {
		// Create a new root node
		root = NewMerkleNode(bt.db, newPos(0, 0), bt.Aggreation)

		// Create a new hash table
		bt.hashTable = NewHashTable(bt.db, bt.Capacity)
	} else {
		if bt.db == nil {
			return ErrDbNotOpen
		}
		rhash := common.BytesToHash(rootHash)
		// Read an existed bucket_tree from db
		root, err = bt.db.GetNode(rhash)
		if err != nil {
			log.Errorf("Failed to get root node:%s", err)
			return err
		}
		bt.hashTable, err = bt.db.GetHashTable(rhash)
		if err != nil {
			log.Errorf("Failed to get hash table from db:%s", err)
			return err
		}
	}
	bt.walkCreateNode(root, 0)
	return nil
}

// Recursive create children node
// {num} the amount of nodes at current level
func (bt *BucketTree) walkCreateNode(curr *MerkleNode, level int) *MerkleNode {
	// Put node to global map of tree
	bt.putNode(curr.Pos, curr)

	num := pow(bt.Aggreation, level)
	if num >= bt.Capacity {
		// leaf node
		if bt.llevel == 0 {
			bt.llevel = level
		}
		curr.leaf = true
	} else {
		var err error
		for i := 0; i < bt.Aggreation; i++ {
			ind := curr.Pos.Index*bt.Aggreation + i
			if hash := curr.Children[i]; !hash.Nil() {
				curr.childNodes[i], err = bt.db.GetNode(hash)
				if err != nil {
					log.Errorf("cannot find node by hash, fatal error")
					curr.childNodes[i] = NewMerkleNode(bt.db, newPos(level+1, ind), bt.Aggreation)
				}
			} else {
				curr.childNodes[i] = NewMerkleNode(bt.db, newPos(level+1, ind), bt.Aggreation)
			}
			bt.walkCreateNode(curr.childNodes[i], level+1)
		}
	}
	return curr
}

func (bt *BucketTree) Prepare(dirty WriteSet) error {
	for k, v := range dirty {
		err := bt.hashTable.put(k, v)
		if err != nil {
			return err
		}
	}
	if len(bt.hashTable.dirty) > 0 {
		bt.dirty = true
	}
	return nil
}

func (bt *BucketTree) Process() (common.Hash, error) {
	err := bt.processNodes()
	if err != nil {
		log.Errorf("Error occur when processing nodes: %s", err)
		return common.Hash{}, err
	}
	root, err := bt.getNode(newPos(0, 0))
	if err != nil {
		return common.Hash{}, err
	}
	return root.computeHash()
}

// Process dirty nodes
func (bt *BucketTree) processNodes() error {
	lowestPos := newPos(bt.llevel, 0)
	for i := range bt.hashTable.dirty {
		lowestPos.Index = i
		leaf, err := bt.getNode(lowestPos)
		if err != nil {
			log.Errorf("Stop processing node: %s", err)
			return err
		}
		bucket := bt.hashTable.buckets[i]
		bucket.computeHash()
		leaf.setHash(bucket.Hash())

		// Collect dirty node
		pos := lowestPos.copy()
		for pos.Level > 0 {
			parentPos := pos.getParent(bt.Aggreation)
			parent, err := bt.getNode(parentPos)
			if err != nil {
				log.Errorf("Stop processing parent node: %s", err)
				return err
			}
			parent.dirty[i%bt.Aggreation] = struct{}{}
			pos = parentPos
			i /= bt.Aggreation
		}
	}
	return nil
}

// commit commits all data in memory to db.Batch.
// At this time, data is not really stored in db, so you should
// explicitly invoke batch.Write().
func (bt *BucketTree) Commit(batch tdb.Batch) error {
	if bt.db == nil {
		return ErrDbNotOpen
	}
	if !bt.dirty {
		return nil
	}
	_, err := bt.Process()
	if err != nil {
		log.Error("Error occurs when processing, stop commit")
		return err
	}
	// Compute bucket hash and put new buckets to db
	err = bt.hashTable.commit(batch)
	if err != nil {
		return err
	}
	root, err := bt.getNode(newPos(0, 0))
	if err != nil {
		return err
	}
	err = bt.db.PutHashTable(batch, root.Hash(), bt.hashTable)
	if err != nil {
		return err
	}
	err = bt.commitNode(batch, root)
	if err != nil {
		return err
	}
	bt.dirty = false
	return nil
}

// Commit node store
func (bt *BucketTree) commitNode(batch tdb.Batch, node *MerkleNode) error {
	if node == nil {
		return nil
	}
	node.commit(batch)
	for i, child := range node.childNodes {
		if _, dirty := node.dirty[i]; dirty {
			err := bt.commitNode(batch, child)
			if err != nil {
				return err
			}
			node.dirty[i] = struct{}{}
		}
	}
	return nil
}

func (bt *BucketTree) Copy() *BucketTree {
	newTree := *bt
	newTree.hashTable = bt.hashTable.copy()
	return &newTree
}

func (bt *BucketTree) Verify(data []byte) {
	// TODO verify data

}

// Purge release the cache memory to avoid mem peak
func (bt *BucketTree) Purge() {
	bt.hashTable.purge()
	bt.node = sync.Map{}
}

// Get data from hash table by key
func (bt *BucketTree) Get(key []byte) ([]byte, error) {
	return bt.hashTable.get(string(key))
}

func pow(a, m int) int {
	if m == 0 {
		return 1
	}
	return a * pow(a, m-1)
}

func Hash(set WriteSet) (common.Hash, error) {
	tree := new(BucketTree)
	tree.Init(nil)
	tree.Prepare(set)
	root, err := tree.Process()
	if err != nil {
		return common.Hash{}, err
	}
	return root, nil
}

func Commit(set WriteSet, db tdb.Database) error {
	tree := NewBucketTree(db)
	tree.Init(nil)
	tree.Prepare(set)
	batch := db.NewBatch()
	if err := tree.Commit(batch); err != nil {
		return err
	}
	return batch.Write()
}
