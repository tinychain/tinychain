package db

import (
	"sync"
)

var (
	batchMgr *BatchMgr
)

// BatchMgr manages db write batch
type BatchMgr struct {
	batches sync.Map
}

func newBatchMgr() *BatchMgr {
	return &BatchMgr{}
}

func (bm *BatchMgr) getBatch(height uint64) Batch {
	if batch, ok := bm.batches.Load(height); ok {
		return batch.(Batch)
	}
	return nil
}

func (bm *BatchMgr) addBatch(height uint64, batch Batch) {
	bm.batches.Store(height, batch)
}

func (bm *BatchMgr) delBatch(height uint64) {
	bm.batches.Delete(height)
}

func GetBatch(db Database, height uint64) Batch {
	if batchMgr == nil {
		batchMgr = newBatchMgr()
	}

	if batch := batchMgr.getBatch(height); batch != nil {
		return batch
	}

	batch := db.NewBatch()
	batchMgr.addBatch(height, batch)
	return batch
}

func CommitBatch(db Database, height uint64) error {
	batch := GetBatch(db, height)
	if err := batch.Write(); err != nil {
		return err
	}
	batchMgr.batches.Delete(height)
	return nil
}
