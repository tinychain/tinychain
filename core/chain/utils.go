package chain

import (
	"sync"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/core/types"
)

var (
	chain *Ledger
	once  sync.Once
)

func initChain(bc *Ledger) {
	once.Do(func() {
		if chain == nil {
			chain = bc
		}
	})

}

func GetHeightOfChain() uint64 {
	if chain == nil {
		return 0
	}
	return chain.LastBlock().Height()
}

func GetBlockByHeight(height uint64) *types.Block {
	if chain == nil {
		return nil
	}
	return chain.GetBlockByHeight(height)
}

func GetBlock(hash common.Hash, height uint64) *types.Block {
	if chain == nil {
		return nil
	}
	return chain.GetBlock(hash, height)
}

func GetBlockByHash(hash common.Hash) *types.Block {
	if chain == nil {
		return nil
	}
	return chain.GetBlockByHash(hash)
}
