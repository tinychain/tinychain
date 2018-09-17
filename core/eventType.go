package core

import (
	"tinychain/core/types"
)

/*
	Blockchain events
 */

/*
	Block events
 */
type NewBlockEvent struct {
	Block *types.Block
}

type BlockBroadcastEvent struct{}

type ExecBlockEvent struct {
	Block *types.Block
}

type ExecFinishEvent struct {
	Res bool // exec result.If success,set true
}

// BlockReadyEvent will be post after block pool received a block and store into pool.
type BlockReadyEvent struct {
	Block *types.Block
}

type ProposeBlockEvent struct {
	Block *types.Block
}

// ConsensusEvent will be posted after a new block proposed by the BP
// completed execution without errors
type ConsensusEvent struct {
	Block *types.Block
}

type CommitBlockEvent struct {
	Block *types.Block
}

type CommitCompleteEvent struct {
	Block *types.Block
}

// NewReceiptsEvent will be posted after a block from other nodes come in,
// completed execution without errors and passed verification.
type NewReceiptsEvent struct {
	Block    *types.Block
	Receipts types.Receipts
}

/*
	Transaction events
 */
type NewTxEvent struct {
	Tx *types.Transaction
}

type NewTxsEvent struct {
	Txs types.Transactions
}

type ExecPendingTxEvent struct {
	Txs types.Transactions
}

type TxBroadcastEvent struct{}

type RollbackEvent struct{}
