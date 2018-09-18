package state

import (
	json "github.com/json-iterator/go"
	"math/big"
	"tinychain/common"
	"tinychain/common/cache"
	"tinychain/core/bmt"
	"tinychain/core/chain"
	"tinychain/db"
)

var (
	emptyCodeHash = common.Sha256(nil)
)

// Value is not actually hash, but just a 32 bytes array
type Storage map[common.Hash][]byte

type stateObject struct {
	sdb     *StateDB
	address common.Address
	data    *Account
	code    []byte     // contract code bytes
	bmt     BucketTree // bucket tree of this account

	cacheStorage    Storage      // storage cache
	lbnCacheStorage *cache.Cache // latest block number storage cache
	dirtyStorage    Storage      // dirty storage

	dirtyCode bool // code is updated or not
	suicided  bool
	deleted   bool

	onDirty func(addr common.Address)
}

type Account struct {
	Nonce    uint64      `json:"nonce"`
	Balance  *big.Int    `json:"balance"`
	Root     common.Hash `json:"root"`
	CodeHash common.Hash `json:"code_hash"`
}

func (s *Account) Serialize() ([]byte, error) {
	return json.Marshal(s)
}

func (s *Account) Deserialize(data []byte) error {
	return json.Unmarshal(data, s)
}

func newStateObject(state *StateDB, address common.Address, data *Account, onDirty func(addr common.Address)) *stateObject {
	return &stateObject{
		sdb:             state,
		address:         address,
		data:            data,
		cacheStorage:    make(Storage),
		lbnCacheStorage: cache.NewCache(cache.NewLBN()),
		dirtyStorage:    make(Storage),
		onDirty:         onDirty,
	}
}

func (s *stateObject) Address() common.Address {
	return s.address
}

func (s *stateObject) Code() []byte {
	return s.code
}

func (s *stateObject) CodeSize() int {
	return len(s.code)
}

func (s *stateObject) Balance() *big.Int {
	return s.data.Balance
}

func (s *stateObject) CodeHash() common.Hash {
	return s.data.CodeHash
}

func (s *stateObject) Root() common.Hash {
	return s.data.Root
}

func (s *stateObject) SetCode(code []byte) {
	s.code = code
	s.data.CodeHash = common.Sha256(code)
	s.dirtyCode = true
	if s.onDirty != nil {
		s.onDirty(s.address)
	}
}

func (s *stateObject) AddBalance(amount *big.Int) {
	s.SetBalance(new(big.Int).Add(s.data.Balance, amount))
	if s.onDirty != nil {
		s.onDirty(s.address)
	}
}

func (s *stateObject) SubBalance(amount *big.Int) {
	if s.Balance().Sign() == 0 {
		return
	}
	s.SetBalance(new(big.Int).Sub(s.data.Balance, amount))
	if s.onDirty != nil {
		s.onDirty(s.address)
	}
}

func (s *stateObject) SetBalance(amount *big.Int) {
	s.data.Balance = amount
	if s.onDirty != nil {
		s.onDirty(s.address)
	}
}

func (s *stateObject) Nonce() uint64 {
	return s.data.Nonce
}

func (s *stateObject) SetNonce(nonce uint64) {
	s.data.Nonce = nonce
	if s.onDirty != nil {
		s.onDirty(s.address)
	}
}

func (s *stateObject) Bmt(db db.Database) BucketTree {
	if tree := s.bmt; tree != nil {
		return tree
	}
	tree := bmt.NewBucketTree(db)
	tree.Init(s.data.Root.Bytes())
	s.bmt = tree
	return tree
}

func (s *stateObject) GetState(key common.Hash) []byte {
	if val, exist := s.cacheStorage[key]; exist {
		return val
	}

	if val, _ := s.lbnCacheStorage.Get(key); val != nil {
		return val.([]byte)
	}

	// Load slot from bucket merkel tree
	val, err := s.bmt.Get(key.Bytes())
	if err != nil {
		return nil
	}
	s.SetState(key, val, chain.GetHeightOfChain())
	return val

}

func (s *stateObject) SetState(key common.Hash, value []byte, blockNum uint64) {
	s.cacheStorage[key] = value
	s.lbnCacheStorage.Add(key, value, blockNum)
	s.dirtyStorage[key] = value
	if s.onDirty != nil {
		s.onDirty(s.address)
	}
}

func (s *stateObject) updateRoot() (common.Hash, error) {
	dirtySet := bmt.NewWriteSet()
	for key, value := range s.dirtyStorage {
		delete(s.dirtyStorage, key)
		dirtySet[key.String()] = value
		// TODO if value.Nil() ?
	}

	if err := s.bmt.Prepare(dirtySet); err != nil {
		return common.Hash{}, err
	}

	return s.bmt.Process()
}

func (s *stateObject) Commit(batch db.Batch) error {
	err := s.bmt.Commit(batch)
	if err != nil {
		return err
	}
	s.data.Root = s.bmt.Hash()
	s.evict()
	return nil
}

func (s *stateObject) deepCopy() *stateObject {
	newAcc := *s.data
	sobj := newStateObject(s.sdb, s.address, &newAcc, s.onDirty)
	sobj.code = s.code
	sobj.dirtyCode = s.dirtyCode
	if tree := s.bmt; tree != nil {
		sobj.bmt = tree.Copy()
	}
	return s
}

func (s *stateObject) empty() bool {
	return s.data.Nonce == 0 && s.data.Balance.Sign() == 0 && s.data.CodeHash == emptyCodeHash
}

func (s *stateObject) markSuicided() {
	s.suicided = true
}

func (s *stateObject) evict() {
	n := s.lbnCacheStorage.EvictWithStrategy(func(blockNum uint64) bool {
		if chain.GetHeightOfChain() < evictBlockGap {
			return false
		}
		return blockNum <= chain.GetHeightOfChain()-evictBlockGap
	})
	log.Debugf("evict %d state value after commit at height #%d", n, chain.GetHeightOfChain())
}
