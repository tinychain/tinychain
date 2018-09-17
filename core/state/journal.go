package state

import (
	"encoding/json"
	"math/big"
	"sync/atomic"
	"tinychain/common"
)

type journalEntry interface {
	undo(*StateDB)
	serialize() ([]byte, error)
}

// journal contains the list of state modifications applied since the last state commit.
type journal struct {
	entries []journalEntry // Current changes tracked by the journal
	//dirties map[common.Address]int // Dirty accounts and the number of changes
}

func newJournal() *journal {
	return &journal{}
}

func (j *journal) append(entry journalEntry) {
	j.entries = append(j.entries, entry)
}

func (j *journal) revert(state *StateDB, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		j.entries[i].undo(state)
	}
	j.entries = j.entries[:snapshot]
}

func (j *journal) delete(index int) {
	j.entries = append(j.entries[:index], j.entries[index+1:]...)
}

func (j *journal) length() int {
	return len(j.entries)
}

type (
	// Changes of the account bucket tree
	createObjectChange struct {
		Account *common.Address `json:"account"`
	}
	resetObjectChange struct {
		prev *stateObject
	}
	suicideChange struct {
		Account     *common.Address `json:"account"`
		prev        bool
		prevBalance *big.Int
	}
	// Changes of the individual accounts
	balanceChange struct {
		Account *common.Address `json:"account"`
		Amount  *big.Int        `json:"amount"`
	}

	gasChange struct {
		Account *common.Address `json:"account"`
		Amount  uint64          `json:"amount"`
	}

	nonceChange struct {
		Account *common.Address `json:"account"`
		Prev    uint64          `json:"prev"`
	}
	storageChange struct {
		Account *common.Address `json:"account"`
		Key     common.Hash     `json:"key"`
		PreVal  []byte          `json:"pre_val"`
		Height  uint64          `json:"height"`
	}
	codeChange struct {
		Account  *common.Address `json:"account"`
		PrevCode []byte          `json:"prev_code"`
		PrevHash common.Hash     `json:"prev_code"`
	}
	logChange struct {
		txHash common.Hash
	}

	// Change of the state value

)

func (ch createObjectChange) undo(s *StateDB) {
	delete(s.stateObjects, *ch.Account)
	delete(s.stateObjectsDirty, *ch.Account)
}

func (ch createObjectChange) serialize() ([]byte, error) {
	return json.Marshal(ch)
}

func (ch resetObjectChange) undo(s *StateDB) {
	s.setStateObj(ch.prev)
}

func (ch resetObjectChange) serialize() ([]byte, error) {
	return json.Marshal(ch)
}

func (ch suicideChange) undo(s *StateDB) {
	obj := s.GetStateObj(*ch.Account)
	if obj != nil {
		obj.suicided = ch.prev
		obj.SetBalance(ch.prevBalance)
	}
}

func (ch suicideChange) serialize() ([]byte, error) {
	return json.Marshal(ch)
}

func (ch balanceChange) undo(s *StateDB) {
	if obj := s.GetStateObj(*ch.Account); obj != nil {
		obj.SubBalance(ch.Amount)
	}
}

func (ch balanceChange) serialize() ([]byte, error) {
	return json.Marshal(ch)
}

func (ch gasChange) undo(s *StateDB) {
	if obj := s.GetStateObj(*ch.Account); obj != nil {
		obj.AddBalance(new(big.Int).SetUint64(ch.Amount))
	}
}

func (ch gasChange) serialize() ([]byte, error) {
	return json.Marshal(ch)
}

func (ch nonceChange) undo(s *StateDB) {
	if obj := s.GetStateObj(*ch.Account); obj != nil {
		obj.SetNonce(ch.Prev)
	}
}

func (ch nonceChange) serialize() ([]byte, error) {
	return json.Marshal(ch)
}

func (ch storageChange) undo(s *StateDB) {
	if obj := s.GetStateObj(*ch.Account); obj != nil {
		obj.SetState(ch.Key, ch.PreVal, ch.Height)
	}
}

func (ch storageChange) serialize() ([]byte, error) {
	return json.Marshal(ch)
}

func (ch codeChange) undo(s *StateDB) {
	if obj := s.GetStateObj(*ch.Account); obj != nil {
		obj.SetCode(ch.PrevCode)
	}
}

func (ch codeChange) serialize() ([]byte, error) {
	return json.Marshal(ch)
}

func (ch logChange) undo(s *StateDB) {
	logs := s.GetLogs(ch.txHash)
	if len(logs) == 1 {
		delete(s.txStat.logs, ch.txHash)
	} else {
		s.txStat.logs[ch.txHash] = logs[:len(logs)-1]
	}
	atomic.AddUint32(&s.logSize, -1)
}

func (ch logChange) serialize() ([]byte, error) {
	return json.Marshal(ch)
}
