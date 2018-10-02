package bmt

import (
	json "github.com/json-iterator/go"
	"sync"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/db"
)

type Position struct {
	Level int `json:"level"` // the level of the node in tree
	Index int `json:"index"` // the index of the node in current level
}

func newPos(level int, index int) *Position {
	if level < 0 || index < 0 {
		return nil
	}
	return &Position{
		level,
		index,
	}
}

func (pos *Position) getParent(aggre int) *Position {
	return newPos(pos.Level-1, pos.Index/aggre)
}

func (pos *Position) copy() *Position {
	return newPos(pos.Level, pos.Index)
}

type MerkleNode struct {
	db         *BmtDB
	H          common.Hash   `json:"hash"`
	Pos        *Position     `json:"pos"`      // position of the node
	Children   []common.Hash `json:"children"` // children hash, for locating in db
	childNodes []*MerkleNode                   // cache node list
	leaf       bool                            // Set true if is leaf
	dirty      map[int]struct{}
	lock       sync.RWMutex
}

func NewMerkleNode(db *BmtDB, pos *Position, aggre int) *MerkleNode {
	return &MerkleNode{
		db:         db,
		Pos:        pos,
		Children:   make([]common.Hash, aggre),
		childNodes: make([]*MerkleNode, aggre),
		dirty:      make(map[int]struct{}),
	}
}

func (node *MerkleNode) Hash() common.Hash {
	return node.H
}

func (node *MerkleNode) setHash(hash common.Hash) {
	node.lock.Lock()
	defer node.lock.Unlock()
	node.H = hash
}

// When node is not leaf
func (node *MerkleNode) computeHash() (common.Hash, error) {
	node.lock.Lock()
	defer node.lock.Unlock()
	if node.leaf {
		return node.Hash(), nil
	}
	var bytes []byte
	for i, childHash := range node.Children {
		var hash []byte
		if _, dirty := node.dirty[i]; dirty {
			child := node.childNodes[i]
			h, err := child.computeHash()
			if err != nil {
				return common.Hash{}, err
			}
			node.Children[i] = h
			hash = h.Bytes()
		} else {
			hash = childHash.Bytes()
		}
		bytes = append(bytes, hash...)
	}
	node.H = common.Sha256(bytes)
	return node.H, nil
}

func (node *MerkleNode) commit(batch db.Batch) error {
	if node.db == nil {
		return ErrDbNotOpen
	}
	return node.db.PutNode(batch, node.Hash(), node)
}

func (node *MerkleNode) serialize() ([]byte, error) {
	return json.Marshal(node)
}

func (node *MerkleNode) deserialize(b []byte) error {
	return json.Unmarshal(b, node)
}
