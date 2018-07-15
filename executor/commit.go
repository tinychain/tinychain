package executor

import (
	"tinychain/core/types"
	"tinychain/common"
)

func (ex *Executor) commit(block *types.Block) error {
	if err := ex.persistTxs(block); err != nil {
		log.Errorf("failed to persist tx metas, err:%s", err)
		return err
	}

	if err := ex.commitBlock(block); err != nil {
		return err
	}

	if _, err := ex.stateCommit(); err != nil {
		log.Errorf("failed to commit state to db, err:%s", err)
		return err
	}
	log.Infof("New block height = #%d commits. Hash = %s", block.Height(), block.Hash().Hex())
	return nil
}

// stateCommit commits the state transition of the current round
func (ex *Executor) stateCommit() (common.Hash, error) {
	return ex.state.Commit()
}

func (ex *Executor) persistTxs(block *types.Block) error {
	return ex.db.PutTxMetas(block.Transactions, block.Hash(), block.Height())
}

func (ex *Executor) persistReceipts(block *types.Block, receipts types.Receipts) error {
	return ex.db.PutReceipts(block.Height(), block.Hash(), receipts)
}

func (ex *Executor) commitBlock(block *types.Block) error {
	return ex.chain.AddBlock(block)
}
