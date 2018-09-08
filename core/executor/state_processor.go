package executor

import (
	"tinychain/common"
	"tinychain/consensus"
	"tinychain/core/chain"
	"tinychain/core/state"
	"tinychain/core/types"
	"tinychain/core/vm"
	"tinychain/core/vm/evm"
)

type vmType int

const (
	EVM vmType = iota
	EWASM
	JSVM
)

type StateProcessor struct {
	conf    *common.Config
	bc      *chain.Blockchain
	statedb *state.StateDB
	engine  consensus.Engine
}

func NewStateProcessor(config *common.Config, bc *chain.Blockchain, statedb *state.StateDB, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		bc:      bc,
		statedb: statedb,
		engine:  engine,
		conf:    config,
	}
}

// Process apply transaction in state
func (sp *StateProcessor) Process(block *types.Block) (types.Receipts, error) {
	var (
		receipts     types.Receipts
		totalGasUsed uint64
		header       = block.Header
	)

	for _, tx := range block.Transactions {
		receipt, gasUsed, err := ApplyTransaction(sp.conf, sp.bc, nil, sp.statedb, header, tx)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
		totalGasUsed += gasUsed
	}
	block.Header.GasUsed = totalGasUsed

	return receipts, nil
}

func ApplyTransaction(cfg *common.Config, bc *chain.Blockchain, author *common.Address, statedb *state.StateDB, header *types.Header, tx *types.Transaction, ) (*types.Receipt, uint64, error) {
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms
	vmenv := newVM(cfg, tx, header, bc, author, statedb)
	// Apply the tx to current state
	_, gasUsed, failed, err := ApplyTx(vmenv, tx)
	if err != nil {
		return nil, 0, err
	}
	// Get intermediate root of current state
	root, err := statedb.IntermediateRoot()
	if err != nil {
		return nil, 0, err
	}
	receipt := &types.Receipt{
		PostState: root,
		Status:    failed,
		TxHash:    tx.Hash(),
		GasUsed:   gasUsed,
	}
	if tx.To.Nil() {
		// Create contract call
		receipt.SetContractAddress(common.CreateAddress(tx.From, tx.Nonce))
	}

	return receipt, gasUsed, nil
}

func newVM(config *common.Config, tx *types.Transaction, header *types.Header, bc *chain.Blockchain, author *common.Address, statedb *state.StateDB) vm.VM {
	vmType := config.GetString("vm.type")
	switch vmType {
	case vm.EVM:
		// create a vm config
		cfg := evm.Config{}
		// Create a new context to be used in the EVM environment
		context := evm.NewEVMContext(tx, header, bc, author)
		return evm.NewEVM(context, statedb, cfg)
	}
}
