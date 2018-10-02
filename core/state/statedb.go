package state

import (
	"fmt"
	"math/big"
	"sync/atomic"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/common/cache"
	"tinychain/core/bmt"
	"tinychain/core/chain"
	"tinychain/core/types"
	tdb "tinychain/db"
)

const (
	evictBlockGap = 1000 // TODO: should be a configurable parameter
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
	Commit(batch tdb.Batch) error
	Get(key []byte) ([]byte, error)
	Copy() *bmt.BucketTree
	Purge()
}

type revision struct {
	id           int
	journalIndex int
}

// txStat wraps the tx info at a certain execution of transaction
type txStat struct {
	txHash, blockHash common.Hash
	txIndex           uint32
	logs              map[common.Hash]types.Logs
	logSize           uint32
}

type StateDB struct {
	db *cacheDB

	nextRevisionId int        // local revision id
	revision       []revision // snapshot version index manager
	journal        *journal   // journal of undo wrapper

	bmt            BucketTree // bucket merkle tree of global state
	bmtCacheHeight uint64     // chain height at which last release of bucket tree cache

	stateObjects      map[common.Address]*stateObject // live state objects
	stateObjectsDirty map[common.Address]struct{}     // dirty state objects
	cacheStateObj     *cache.Cache

	currHeight uint64 // block height of process currently
	txStat
}

func New(db tdb.Database, root []byte) (*StateDB, error) {
	if db == nil {
		ldb, err := tdb.NewLDBDataBase("/data/temp")
		if err != nil {
			return nil, err
		}
		db = ldb
	}
	tree := bmt.NewBucketTree(db)
	if err := tree.Init(root); err != nil {
		log.Errorf("Failed to init bucket tree when new state db, %s", err)
		return nil, err
	}
	return &StateDB{
		db:                newCacheDB(db),
		journal:           newJournal(),
		bmt:               tree,
		bmtCacheHeight:    chain.GetHeightOfChain(),
		stateObjects:      make(map[common.Address]*stateObject),
		stateObjectsDirty: make(map[common.Address]struct{}),
		cacheStateObj:     cache.NewCache(cache.NewLBN()),
		txStat: txStat{
			logs: make(map[common.Hash]types.Logs),
		},
	}, nil
}

// Get state object from cache and bucket tree
// If error, return nil
func (sdb *StateDB) GetStateObj(addr common.Address) *stateObject {
	if stateObj, exist := sdb.stateObjects[addr]; exist {
		if stateObj.deleted {
			return nil
		}
		return stateObj
	}

	if val, _ := sdb.cacheStateObj.Get(addr); val != nil {
		stateObj := val.(*stateObject)
		sdb.stateObjects[addr] = stateObj
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
	stateObj := newStateObject(sdb, addr, account, sdb.setDirty)
	code, err := sdb.db.GetCode(account.CodeHash)
	if err == nil {
		stateObj.SetCode(code)
	}
	sdb.setStateObj(stateObj)
	return stateObj
}

// CreateAccount explicitly creates a state object. If a state object with the address
// already exists the balance is carried over to the new account.
//
// CreateAccount is called during the EVM CREATE operation. The situation might arise that
// a contract does the following:
//
//   1. sends funds to sha(account ++ (nonce + 1))
//   2. tx_create(sha(account ++ nonce)) (note that this gets the address of 1)
//
// Carrying over the balance ensures that Ether doesn't disappear.
func (sdb *StateDB) CreateAccount(addr common.Address) {
	newObj, prev := sdb.createStateObj(addr)
	if prev != nil {
		newObj.SetBalance(prev.Balance())
	}
}

func (sdb *StateDB) createStateObj(addr common.Address) (*stateObject, *stateObject) {
	prev := sdb.GetStateObj(addr)
	account := &Account{
		Nonce:   uint64(0),
		Balance: new(big.Int),
	}
	newObj := newStateObject(sdb, addr, account, sdb.setDirty)
	if prev == nil {
		sdb.journal.append(createObjectChange{&addr})
	} else {
		sdb.journal.append(resetObjectChange{prev})
	}
	sdb.setStateObj(newObj)
	return newObj, prev
}

func (sdb *StateDB) UpdateCurrHeight(height uint64) {
	atomic.StoreUint64(&sdb.currHeight, height)
}

// Set "live" state object
func (sdb *StateDB) setStateObj(object *stateObject) {
	sdb.stateObjects[object.Address()] = object
	sdb.cacheStateObj.Add(object.Address(), object, chain.GetHeightOfChain())
}

func (sdb *StateDB) setDirty(addr common.Address) {
	sdb.stateObjectsDirty[addr] = struct{}{}
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
		sdb.journal.append(storageChange{
			Account: &addr,
			Key:     key,
			PreVal:  stateObj.GetState(key),
			Height:  sdb.currHeight,
		})
		stateObj.SetState(key, value, sdb.currHeight)
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
	if stateObj == nil || stateObj.deleted {
		stateObj, _ = sdb.createStateObj(addr)
	}
	return stateObj
}

func (sdb *StateDB) GetNonce(addr common.Address) uint64 {
	stateObj := sdb.GetStateObj(addr)
	if stateObj != nil {
		return stateObj.Nonce()
	}
	return 0
}

func (sdb *StateDB) SetNonce(addr common.Address, nonce uint64) {
	stateObj := sdb.GetOrNewStateObj(addr)
	if stateObj != nil {
		sdb.journal.append(nonceChange{
			Account: &addr,
			Prev:    stateObj.Nonce(),
		})
		stateObj.SetNonce(nonce)
	}
}

func (sdb *StateDB) GetBalance(addr common.Address) *big.Int {
	stateObj := sdb.GetStateObj(addr)
	if stateObj != nil {
		return stateObj.Balance()
	}
	return nil
}

func (sdb *StateDB) SetBalance(addr common.Address, target *big.Int) {
	stateObj := sdb.GetOrNewStateObj(addr)
	if stateObj != nil {
		prev := stateObj.Balance()
		sdb.journal.append(balanceChange{
			Account: &addr,
			Amount:  prev.Sub(target, prev),
		})
		stateObj.SetBalance(target)
	}
}

func (sdb *StateDB) AddBalance(addr common.Address, amount *big.Int) {
	stateObj := sdb.GetOrNewStateObj(addr)
	if stateObj != nil {
		sdb.journal.append(balanceChange{
			Account: &addr,
			Amount:  amount,
		})
		stateObj.AddBalance(amount)
	}
}

func (sdb *StateDB) SubBalance(addr common.Address, amount *big.Int) {
	stateObj := sdb.GetOrNewStateObj(addr)
	if stateObj != nil {
		sdb.journal.append(balanceChange{
			Account: &addr,
			Amount:  amount.Mul(amount, new(big.Int).SetInt64(-1)),
		})
		stateObj.SubBalance(amount)
	}
}

// ChargeGas charge a given address as gas during the transaction execution,
// but will give back at the state commit process.
// This mechanism is make free transactions possible and defend sending huge amount of transactions
// from the same address viciously.
func (sdb *StateDB) ChargeGas(addr common.Address, amount uint64) {
	stateObj := sdb.GetStateObj(addr)
	if stateObj != nil {
		sdb.journal.append(gasChange{
			Account: &addr,
			Amount:  amount,
		})
		stateObj.SubBalance(new(big.Int).SetUint64(amount))
	}
}

func (sdb *StateDB) GetCode(addr common.Address) []byte {
	if obj := sdb.GetStateObj(addr); obj != nil {
		return obj.Code()
	}
	return nil
}

func (sdb *StateDB) GetCodeSize(address common.Address) int {
	if obj := sdb.GetStateObj(address); obj != nil {
		return obj.CodeSize()
	}
	return 0
}

func (sdb *StateDB) SetCode(addr common.Address, code []byte) {
	stateObj := sdb.GetOrNewStateObj(addr)
	if stateObj != nil {
		sdb.journal.append(codeChange{
			Account:  &addr,
			PrevCode: stateObj.Code(),
			PrevHash: stateObj.CodeHash(),
		})
		stateObj.SetCode(code)
	}
}

func (sdb *StateDB) Exist(addr common.Address) bool {
	s := sdb.GetStateObj(addr)
	return s != nil
}

func (sdb *StateDB) Suicide(addr common.Address) bool {
	obj := sdb.GetStateObj(addr)
	if obj == nil {
		return false
	}

	sdb.journal.append(suicideChange{
		Account:     &addr,
		prev:        obj.suicided,
		prevBalance: obj.Balance(),
	})
	obj.markSuicided()
	obj.SetBalance(new(big.Int))
	return true
}

func (sdb *StateDB) HasSuicided(addr common.Address) bool {
	stateObj := sdb.GetStateObj(addr)
	if stateObj != nil {
		return stateObj.suicided
	}
	return false
}

func (sdb *StateDB) Empty(addr common.Address) bool {
	obj := sdb.GetStateObj(addr)
	if obj != nil {
		return obj.Balance().Cmp(new(big.Int).SetInt64(0)) == 0 && obj.Nonce() == 0 && len(obj.Code()) == 0
	}
	return true
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

func (sdb *StateDB) Commit(batch tdb.Batch) (common.Hash, error) {
	sdb.refundGas() // refund all gas
	dirtySet := bmt.NewWriteSet()

	for addr, stateObj := range sdb.stateObjects {
		_, isDirty := sdb.stateObjectsDirty[addr]
		switch {
		case stateObj.suicided || (isDirty && stateObj.empty()):
			// Delete stateObject
			stateObj.deleted = true
			dirtySet[addr.String()] = nil
			sdb.cacheStateObj.Delete(addr)
		case isDirty:
			stateobj := sdb.stateObjects[addr]
			// Put account data to dirtySet to update world state tree
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
		delete(sdb.stateObjectsDirty, addr)
	}

	if err := sdb.bmt.Prepare(dirtySet); err != nil {
		return common.Hash{}, err
	}
	if err := sdb.bmt.Commit(batch); err != nil {
		return common.Hash{}, err
	}
	sdb.reset() // reset state object mem cache
	sdb.clearJournal()
	return sdb.bmt.Hash(), nil
}

// refund applies balanceChange journals and give back all gas to original senders.
func (sdb *StateDB) refundGas() {
	for i := 0; i < sdb.journal.length(); i++ {
		change := sdb.journal.entries[i]
		if gasChange, ok := change.(gasChange); ok {
			gasChange.undo(sdb)
			sdb.journal.delete(i)
			i--
		}
	}
}

func (sdb *StateDB) reset() {
	sdb.stateObjects = make(map[common.Address]*stateObject)
	sdb.stateObjectsDirty = make(map[common.Address]struct{})
	// evict old cache
	sdb.cacheStateObj.EvictWithStrategy(func(blockNum uint64) bool {
		if sdb.currHeight < evictBlockGap {
			return false
		}
		return blockNum < sdb.currHeight-evictBlockGap
	})
}

// Snapshot returns an identifier for the current revision of the state.
func (sdb *StateDB) Snapshot() int {
	id := sdb.nextRevisionId
	sdb.nextRevisionId++
	sdb.revision = append(sdb.revision, revision{id, sdb.journal.length()})
	return id
}

func (sdb *StateDB) RevertToSnapshot(revid int) {
	// Find the snapshot in the current revision
	var idx int
	for i, revision := range sdb.revision {
		if revision.id >= revid {
			idx = i
			break
		}
	}

	if idx == len(sdb.revision) || sdb.revision[idx].id != revid {
		panic(fmt.Sprintf("revision id %v cannot be reverted", revid))
	}

	snapshot := sdb.revision[idx].journalIndex

	// Replay the journal to undo changes and remove invalid snapshots
	sdb.journal.revert(sdb, snapshot)
	sdb.revision = sdb.revision[:revid]
	sdb.nextRevisionId = revid + 1
}

func (sdb *StateDB) clearJournal() {
	sdb.journal = newJournal()
	sdb.revision = sdb.revision[:0]
}

// Prepare will be called when execute a new tx in state processor
func (sdb *StateDB) Prepare(txHash, blockHash common.Hash, txIndex uint32) {
	sdb.txHash = txHash
	sdb.blockHash = blockHash
	sdb.txIndex = txIndex
}

func (sdb *StateDB) AddLog(log *types.Log) {
	sdb.journal.append(logChange{sdb.txHash})

	log.TxHash = sdb.txHash
	log.BlockHash = sdb.blockHash
	logs := sdb.GetLogs(sdb.txHash)
	sdb.logs[sdb.txHash] = append(logs, log)
	atomic.AddUint32(&sdb.logSize, 1)
}

func (sdb *StateDB) GetLogs(txHash common.Hash) types.Logs {
	return sdb.txStat.logs[txHash]
}
