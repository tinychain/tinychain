package executor

import (
	"tinychain/common"
	"math/big"
	"time"
	"tinychain/core/state"
	tdb "tinychain/db"
	"tinychain/core/types"
)

func (ex *Executor) createGenesis() (*types.Block, error) {
	statedb, err := state.New(ex.db.LDB(), nil)
	if err != nil {
		log.Errorf("failed to init state, %s", err)
		return nil, err
	}
	stateRoot, err := statedb.Commit(tdb.GetBatch(ex.db.LDB(), 0))
	if err != nil {
		log.Errorf("compute state root failed, %s", err)
		return nil, err
	}
	genesis := types.NewBlock(
		&types.Header{
			ParentHash: common.Sha256(common.Hash{}.Bytes()),
			Height:     0,
			StateRoot:  stateRoot,
			Coinbase:   common.HexToAddress("0x0000"),
			Time:       new(big.Int).SetInt64(int64(time.Now().UnixNano())),
			Extra:      []byte("everyone can hold its asset in the blockchain"),
		},
		nil)

	if err := ex.chain.SetGenesis(genesis); err != nil {
		log.Errorf("failed to set genesis to blockchain, %s", err)
		return nil, err
	}
	return genesis, nil
}
