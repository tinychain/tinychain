package state

import (
	"tinychain/common"
	"math/big"
	"encoding/json"
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

func (j *journal) length() int {
	return len(j.entries)
}

type (
	// Changes of the account bucket tree
	createObjectChange struct {
		Account *common.Address `json:"account"`
	}
	suicideChange struct {
		Account     *common.Address `json:"account"`
		prev        bool
		prevBalance *big.Int
	}

	// Changes of the individual accounts
	balanceChange struct {
		Account *common.Address `json:"account"`
		Prev    *big.Int        `json:"prev"`
	}
	nonceChange struct {
		Account *common.Address `json:"account"`
		Prev    uint64          `json:"prev"`
	}
	storageChange struct {
		Account *common.Address `json:"account"`
		Key     common.Hash     `json:"key"`
		PreVal  []byte          `json:"pre_val"`
	}
	codeChange struct {
		Account  *common.Address `json:"account"`
		PrevCode []byte          `json:"prev_code"`
		PrevHash common.Hash     `json:"prev_code"`
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
		obj.SetBalance(ch.Prev)
	}
}

func (ch balanceChange) serialize() ([]byte, error) {
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
		obj.SetState(ch.Key, ch.PreVal)
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
