package state

import (
	"tinychain/common"
	"tinychain/bmt"
	"tinychain/db/leveldb"
	"math/big"
)

var (
	log = common.GetLogger("state")
)

// Bucket tree
type BucketTree interface {
	Hash() common.Hash
	Init(root []byte) error
	Prepare(dirty bmt.WriteSet) error
	Process() (common.Hash, error)
	Commit(batch *leveldb.Batch) error
	Get(key []byte) ([]byte, error)
	Copy() *bmt.BucketTree
}

type StateDB struct {
	db                *cacheDB
	bmt               BucketTree                      // bucket merkle tree of global state
	stateObjects      map[common.Address]*stateObject // live state objects
	stateObjectsDirty map[common.Address]struct{}     // dirty state objects
}

func New(db *leveldb.LDBDatabase, root []byte) (*StateDB, error) {
	tree := bmt.NewBucketTree(db)
	if err := tree.Init(root); err != nil {
		log.Errorf("Failed to init bucket tree when new state db, %s", err)
		return nil, err
	}
	return &StateDB{
		db:                newCacheDB(db),
		bmt:               tree,
		stateObjects:      make(map[common.Address]*stateObject),
		stateObjectsDirty: make(map[common.Address]struct{}),
	}, nil
}

// Get state object from cache and bucket tree
// If error, return nil
func (sdb *StateDB) GetStateObj(addr common.Address) *stateObject {
	if stateObj, exist := sdb.stateObjects[addr]; exist {
		return stateObj
	}
	data, err := sdb.bmt.Get(addr.Bytes())
	if err != nil {
		return nil
	}
	account := &Account{}
	err = account.Deserialize(data)
	if err != nil {
		return nil
	}
	stateObj := newStateObject(addr, account)
	code, _ := sdb.db.GetCode(account.CodeHash)
	if code != nil {
		stateObj.SetCode(code)
	}
	sdb.setStateObj(stateObj)
	return stateObj
}

// Create a new state object
func (sdb *StateDB) CreateStateObj(addr common.Address) *stateObject {
	account := &Account{
		Nonce:   uint64(0),
		Balance: new(big.Int),
	}
	newObj := newStateObject(addr, account)
	sdb.setStateObj(newObj)
	return newObj
}

// Set "live" state object
func (sdb *StateDB) setStateObj(object *stateObject) {
	sdb.stateObjects[object.Address()] = object
	sdb.stateObjectsDirty[object.Address()] = struct{}{}
}

// Get state of an account with address
func (sdb *StateDB) GetState(addr common.Address, key common.Hash) []byte {
	stateObj := sdb.GetStateObj(addr)
	if stateObj != nil {
		return stateObj.GetState(key)
	}
	return nil
}

// Set state of an account
func (sdb *StateDB) SetState(addr common.Address, key common.Hash, value []byte) {
	stateObj := sdb.GetOrNewStateObj(addr)
	if stateObj != nil {
		stateObj.SetState(key, value)
	}
}

func (sdb *StateDB) GetCodeHash(addr common.Address) common.Hash {
	stateObj := sdb.GetStateObj(addr)
	if stateObj != nil {
		return stateObj.CodeHash()
	}
	return common.Hash{}
}

// Get state bucket merkel tree of state object
func (sdb *StateDB) StateBmt(addr common.Address) BucketTree {
	stateObj := sdb.GetStateObj(addr)
	if stateObj != nil {
		return stateObj.bmt.Copy()
	}
	return nil
}

// Get or create a state object
func (sdb *StateDB) GetOrNewStateObj(addr common.Address) *stateObject {
	stateObj := sdb.GetStateObj(addr)
	if stateObj != nil {
		return sdb.CreateStateObj(addr)
	}
	return stateObj
}

func (sdb *StateDB) SetBalance(addr common.Address, amount *big.Int) {
	stateObj := sdb.GetOrNewStateObj(addr)
	if stateObj != nil {
		stateObj.SetBalance(amount)
	}
}

func (sdb *StateDB) AddBalance(addr common.Address, amount *big.Int) {
	stateObj := sdb.GetOrNewStateObj(addr)
	if stateObj != nil {
		stateObj.AddBalance(amount)
	}
}

func (sdb *StateDB) SubBalance(addr common.Address, amount *big.Int) {
	stateObj := sdb.GetOrNewStateObj(addr)
	if stateObj != nil {
		stateObj.SubBalance(amount)
	}
}

func (sdb *StateDB) SetNonce(addr common.Address, nonce uint64) {
	stateObj := sdb.GetOrNewStateObj(addr)
	if stateObj != nil {
		stateObj.SetNonce(nonce)
	}
}

func (sdb *StateDB) SetCode(addr common.Address, code []byte) {
	stateObj := sdb.GetOrNewStateObj(addr)
	if stateObj != nil {
		stateObj.SetCode(code)
	}
}

func (sdb *StateDB) Exist(addr common.Address) bool {
	s := sdb.GetStateObj(addr)
	return s != nil
}

// Process dirty state object to state tree and get intermediate root
func (sdb *StateDB) IntermediateRoot() (common.Hash, error) {
	dirtySet := bmt.NewWriteSet()
	for addr := range sdb.stateObjectsDirty {
		stateobj := sdb.stateObjects[addr]
		data, _ := stateobj.data.Serialize()
		dirtySet[addr.String()] = data
	}
	if err := sdb.bmt.Prepare(dirtySet); err != nil {
		return common.Hash{}, err
	}
	return sdb.bmt.Process()
}

func (sdb *StateDB) Commit(batch *leveldb.Batch) (common.Hash, error) {
	dirtySet := bmt.NewWriteSet()

	for addr := range sdb.stateObjectsDirty {
		delete(sdb.stateObjectsDirty, addr)
		stateobj := sdb.stateObjects[addr]
		// Put account data to dirtySet
		data, _ := stateobj.data.Serialize()
		dirtySet[addr.String()] = data

		// Put code bytes to codeSet
		if stateobj.dirtyCode {
			if err := sdb.db.PutCode(stateobj.CodeHash(), stateobj.Code()); err != nil {
				stateobj.dirtyCode = false
			}
		}
		if err := stateobj.Commit(batch); err != nil {
			return common.Hash{}, err
		}
	}

	if err := sdb.bmt.Prepare(dirtySet); err != nil {
		return common.Hash{}, err
	}
	if err := sdb.bmt.Commit(batch); err != nil {
		return common.Hash{}, err
	}
	return sdb.bmt.Hash(), nil
}
