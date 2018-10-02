package vm

import (
	"math/big"
	"github.com/tinychain/tinychain/common"
)

const (
	EVM   = "EVM"
	EWASM = "EWASM"
	JSVM  = "JSVM"
)

type ContractRef interface {
	Address() common.Address
}

// Context represents the context of a virtual machine,
// which should be defined at each implementation of VM.
type Context interface{}

type VM interface {
	DB() StateDB
	Create(caller ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error)
	Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error)
	Coinbase() common.Address
}

type vmConfig interface{}
