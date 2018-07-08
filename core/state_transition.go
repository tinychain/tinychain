package core

import (
	"math/big"
	"tinychain/core/vm"
	"errors"
	"tinychain/core/types"
)

var (
	ErrNonceTooHight    = errors.New("nonce too hight")
	ErrNonceTooLow      = errors.New("nonce too low")
	ErrBalanceNotEnough = errors.New("balance not enough")

	MaxGas = uint64(9999999) // Maximum
)

type StateTransition struct {
	tx      *types.Transaction // state transition event
	evm     *vm.EVM
	statedb vm.StateDB
}

func NewStateTransition(evm *vm.EVM, tx *types.Transaction) *StateTransition {
	return &StateTransition{
		evm:     evm,
		tx:      tx,
		statedb: evm.StateDB,
	}
}

// Make state transition by applying a new event
func ApplyTx(evm *vm.EVM, tx *types.Transaction) ([]byte, uint64, bool, error) {
	return NewStateTransition(evm, tx).Process()
}

// Check nonce is correct or not
// nonce should be equal to that of state object
func (st *StateTransition) preCheck() error {
	nonce := st.statedb.GetNonce(st.tx.From)
	if nonce < st.tx.Nonce {
		return ErrNonceTooHight
	} else if nonce > st.tx.Nonce {
		return ErrNonceTooLow
	}
	return nil
}

func (st *StateTransition) from() vm.AccountRef {
	addr := st.tx.From
	if !st.statedb.Exist(addr) {
		st.statedb.CreateAccount(addr)
	}
	return vm.AccountRef(addr)
}

func (st *StateTransition) to() vm.AccountRef {
	if st.tx == nil {
		return vm.AccountRef{}
	}

	if st.tx.To.Nil() {
		return vm.AccountRef{}
	}
	to := st.tx.To
	//if !st.statedb.Exist(to) {
	//	st.statedb.CreateAccount(to)
	//}
	return vm.AccountRef(to)
}

func (st *StateTransition) data() []byte {
	return st.tx.Payload
}

func (st *StateTransition) gas() uint64 {
	return st.tx.GasLimit
}

func (st *StateTransition) gasPrice() uint64 {
	return st.tx.GasPrice
}

func (st *StateTransition) value() *big.Int {
	return st.tx.Value
}

// Make state transition according to transaction event
func (st *StateTransition) Process() ([]byte, uint64, bool, error) {
	if err := st.preCheck(); err != nil {
		return nil, 0, false, err
	}

	var (
		vmerr   error
		ret     []byte
		leftGas uint64
	)
	if (st.to() == vm.AccountRef{}) {
		// Contract create
		ret, _, leftGas, vmerr = st.evm.Create(st.to(), st.data(), st.gas(), st.value())
	} else {
		// Call contract
		st.statedb.SetNonce(st.from().Address(), st.statedb.GetNonce(st.from().Address())+1)
		ret, leftGas, vmerr = st.evm.Call(st.from(), st.to().Address(), st.data(), st.gas(), st.value())
	}
	if vmerr != nil {
		log.Errorf("VM returned with error %s", vmerr)
		if vmerr == vm.ErrInsufficientBalance {
			return nil, 0, false, vmerr
		}
	}
	gasUsed := st.gas() - leftGas
	//st.statedb.SubBalance(st.from().Address(), new(big.Int).SetUint64(gasUsed))
	//st.statedb.AddBalance(st.evm.Coinbase, new(big.Int).SetUint64(gasUsed))
	balance := st.statedb.GetBalance(st.from().Address())
	if balance.Cmp(new(big.Int).SetUint64(gasUsed*st.gasPrice())) < 0 {
		// TODO:balance not enough
		log.Errorf("balance not enough for transaction %s", st.tx.Hash().Hex())
		return nil, gasUsed, false, ErrBalanceNotEnough
	}

	return ret, gasUsed, vmerr != nil, nil
}
