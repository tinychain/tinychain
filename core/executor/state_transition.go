package executor

import (
	"errors"
	"math/big"
	"tinychain/core/types"
	"tinychain/core/vm"
	"tinychain/core/vm/evm"
)

var (
	errNonceTooHight    = errors.New("nonce too hight")
	errBalanceNotEnough = errors.New("balance not enough")

	MaxGas = uint64(9999999) // Maximum
)

type StateTransition struct {
	tx      *types.Transaction // state transition event
	vm      vm.VM
	statedb vm.StateDB
}

func NewStateTransition(virtualMachine vm.VM, tx *types.Transaction) *StateTransition {
	return &StateTransition{
		vm:      virtualMachine,
		tx:      tx,
		statedb: virtualMachine.DB(),
	}
}

// Make state transition by applying a new event
func ApplyTx(virtualMachine vm.VM, tx *types.Transaction) ([]byte, uint64, bool, error) {
	return NewStateTransition(virtualMachine, tx).Process()
}

// Check nonce is correct or not
// nonce should be equal to that of state object
func (st *StateTransition) preCheck() error {
	nonce := st.statedb.GetNonce(st.tx.From)
	if nonce < st.tx.Nonce {
		return errNonceTooHight
	} else if nonce > st.tx.Nonce {
		return errNonceTooLow
	}
	return nil
}

func (st *StateTransition) from() evm.AccountRef {
	addr := st.tx.From
	if !st.statedb.Exist(addr) {
		st.statedb.CreateAccount(addr)
	}
	return evm.AccountRef(addr)
}

func (st *StateTransition) to() evm.AccountRef {
	if st.tx == nil {
		return evm.AccountRef{}
	}

	if st.tx.To.Nil() {
		return evm.AccountRef{}
	}
	to := st.tx.To
	//if !st.statedb.Exist(to) {
	//	st.statedb.CreateAccount(to)
	//}
	return evm.AccountRef(to)
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
	if (st.to() == evm.AccountRef{}) {
		// Contract create
		ret, _, leftGas, vmerr = st.vm.Create(st.to(), st.data(), st.gas(), st.value())
	} else {
		// Call contract
		st.statedb.SetNonce(st.from().Address(), st.statedb.GetNonce(st.from().Address())+1)
		ret, leftGas, vmerr = st.vm.Call(st.from(), st.to().Address(), st.data(), st.gas(), st.value())
	}
	if vmerr != nil {
		log.Errorf("VM returned with error %s", vmerr)
		if vmerr == evm.ErrInsufficientBalance {
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
		return nil, gasUsed, false, errBalanceNotEnough
	}

	return ret, gasUsed, vmerr != nil, nil
}
