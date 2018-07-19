package state

import (
	"tinychain/common"
	"math/big"
	json "github.com/json-iterator/go"
	"tinychain/db/leveldb"
	"tinychain/bmt"
)

// Value is not actually hash, but just a 32 bytes array
type Storage map[common.Hash][]byte

type stateObject struct {
	address common.Address
	data    *Account
	code    []byte     // contract code bytes
	bmt     BucketTree // bucket tree of this account

	cacheStorage Storage // storage cache
	dirtyStorage Storage // dirty storage

	dirtyCode bool // code is updated or not
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

func newStateObject(address common.Address, data *Account) *stateObject {
	return &stateObject{
		address:      address,
		data:         data,
		cacheStorage: make(Storage),
		dirtyStorage: make(Storage),
	}
}

func (s *stateObject) Address() common.Address {
	return s.address
}

func (s *stateObject) Code() []byte {
	return s.code
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
}

func (s *stateObject) AddBalance(amount *big.Int) {
	s.SetBalance(new(big.Int).Add(s.data.Balance, amount))
}

func (s *stateObject) SubBalance(amount *big.Int) {
	s.SetBalance(new(big.Int).Sub(s.data.Balance, amount))
}

func (s *stateObject) SetBalance(amount *big.Int) {
	s.data.Balance = amount
}

func (s *stateObject) Nonce() uint64 {
	return s.data.Nonce
}

func (s *stateObject) SetNonce(nonce uint64) {
	s.data.Nonce = nonce
}

func (s *stateObject) Bmt(db *leveldb.LDBDatabase) BucketTree {
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
	// Load slot from bucket merkel tree
	val, err := s.bmt.Get(key.Bytes())
	if err != nil {
		return nil
	}
	s.SetState(key, val)
	return val

}

func (s *stateObject) SetState(key common.Hash, value []byte) {
	s.cacheStorage[key] = value
	s.dirtyStorage[key] = value
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

func (s *stateObject) Commit(batch *leveldb.Batch) error {
	err := s.bmt.Commit(batch)
	if err != nil {
		return err
	}
	s.data.Root = s.bmt.Hash()
	return nil
}

func (s *stateObject) deepCopy() *stateObject {
	newAcc := *s.data
	sobj := newStateObject(s.address, &newAcc)
	sobj.code = s.code
	sobj.dirtyCode = s.dirtyCode
	if tree := s.bmt; tree != nil {
		sobj.bmt = tree.Copy()
	}
	return s
}
