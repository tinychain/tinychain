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

type BlockReadyEvent struct {
	Height uint64
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
	Height uint64
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
	Height   uint64
	Receipts types.Receipts
}
