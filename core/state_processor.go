package core

import (
	"tinychain/core/types"
	"tinychain/core/state"
	"tinychain/common"
	"tinychain/core/vm"
)

type StateProcessor struct {
	bc      *Blockchain
	statedb *state.StateDB
}

func NewStateProcessor(bc *Blockchain, statedb *state.StateDB) *StateProcessor {
	return &StateProcessor{
		bc:      bc,
		statedb: statedb,
	}
}

// Process apply transaction in state
func (sp *StateProcessor) Process(block *types.Block) (types.Receipts, error) {
	var (
		receipts types.Receipts
		header   = block.Header
	)

	for _, tx := range block.Transactions {
		receipt, err := ApplyTransaction(sp.bc, nil, sp.statedb, header, tx)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}
	// TODO Finalize block
	return receipts, nil
}

func ApplyTransaction(bc *Blockchain, author *common.Address, statedb *state.StateDB, header *types.Header, tx *types.Transaction) (*types.Receipt, error) {
	// Create a new context to be used in the EVM environment
	context := NewEVMContext(tx, header, bc, author)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms
	vmenv := vm.NewEVM(context, statedb, nil, nil)
	// Apply the tx to current state
	_, gasUsed, failed, err := ApplyTx(vmenv, tx)
	if err != nil {
		return nil, err
	}
	// Get intermediate root of current state
	root, err := statedb.IntermediateRoot()
	if err != nil {
		return nil, err
	}
	receipt := types.NewRecipet(root, failed, tx.Hash(), gasUsed)
	if tx.To.Nil() {
		// Create contract call
		receipt.SetContractAddress(common.CreateAddress(tx.From, tx.Nonce))
	}

	return receipt, nil
}
