package executor

import (
	"tinychain/core/types"
	"tinychain/common"
	"tinychain/db"
	"tinychain/core"
)

func (ex *Executor) commit(block *types.Block) error {
	if err := ex.persistTxs(block); err != nil {
		log.Errorf("failed to persist tx metas, err:%s", err)
		return err
	}

	if receipts, exist := ex.receiptsCache.Load(block.Height()); exist {
		err := ex.persistReceipts(block, receipts.(types.Receipts))
		if err != nil {
			log.Errorf("failed to persist receipts, err:%s", err)
			return err
		}
		ex.receiptsCache.Delete(block.Height())
	}

	if _, err := ex.stateCommit(block.Height()); err != nil {
		log.Errorf("failed to put state in batch, err:%s", err)
		return err
	}

	if err := ex.commitBlock(block); err != nil {
		return err
	}

	// Commit data in batch
	if err := db.CommitBatch(ex.db.LDB(), block.Height()); err != nil {
		log.Errorf("failed to commit db.Batch, err:%s", err)
		return err
	}
	log.Infof("New block height = #%d commits. Hash = %s", block.Height(), block.Hash().Hex())
	go ex.event.Post(&core.CommitCompleteEvent{
		Block: block,
	})
	return nil
}

// stateCommit commits the state transition at the given block height
func (ex *Executor) stateCommit(height uint64) (common.Hash, error) {
	return ex.state.Commit(db.GetBatch(ex.db.LDB(), height))
}

func (ex *Executor) persistTxs(block *types.Block) error {
	return ex.db.PutTxMetas(db.GetBatch(ex.db.LDB(), block.Height()), block.Transactions, block.Hash(), block.Height(), false, false)
}

func (ex *Executor) persistReceipts(block *types.Block, receipts types.Receipts) error {
	return ex.db.PutReceipts(db.GetBatch(ex.db.LDB(), block.Height()), block.Height(), block.Hash(), receipts, false, false)
}

func (ex *Executor) commitBlock(block *types.Block) error {
	return ex.chain.CommitBlock(block)
}
