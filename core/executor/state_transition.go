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
	gasPool *GasPool
}

func NewStateTransition(virtualMachine vm.VM, tx *types.Transaction) *StateTransition {
	return &StateTransition{
		vm:      virtualMachine,
		tx:      tx,
		statedb: virtualMachine.DB(),
		gasPool: new(GasPool),
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
	return st.buyGas()
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

	to := st.tx.To
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

func (st *StateTransition) buyGas() error {
	maxGasUsed := st.gas() * st.gasPrice()
	if balance := st.statedb.GetBalance(st.from().Address()); balance.Cmp(new(big.Int).SetUint64(maxGasUsed)) < 0 {
		log.Errorf("balance not enough for transaction %s", st.tx.Hash().Hex())
		return errBalanceNotEnough
	}
	st.statedb.ChargeGas(st.tx.From, maxGasUsed)
	st.gasPool.AddGas(maxGasUsed)
	return nil
}

// refundGas gives the remaining gas in gasPool back to the account of sender
func (st *StateTransition) refundGas() {
	st.statedb.AddBalance(st.from().Address(), new(big.Int).SetUint64(st.gasPool.Gas()))
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
	if st.to().Address().Nil() {
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
	st.gasPool.SubGas(gasUsed * st.gasPrice())
	st.refundGas()

	return ret, gasUsed, vmerr != nil, nil
}
