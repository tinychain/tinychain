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

type BlockCommitEvent struct {
	Height uint64
}

type ExecBlockEvent struct {
	Block *types.Block
}

type ExecFinishEvent struct {
	Res bool // exec result.If success,set true
}

/*
	Transaction events
 */
type NewTxEvent struct {
	Tx *types.Transaction
}

type ExecPendingTxEvent struct {
	Txs types.Transactions
}

type TxBroadcastEvent struct{}
