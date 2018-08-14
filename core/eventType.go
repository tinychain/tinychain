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

type ConsensusEvent struct {
	Block *types.Block
}

type CommitBlockEvent struct {
	Block *types.Block
}

type CommitCompleteEvent struct {
	Block *types.Block
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

/*
	Receipts events
 */
type NewReceiptsEvent struct {
	Block    *types.Block
	Receipts types.Receipts
}
