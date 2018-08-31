package chain

import (
	"tinychain/core/types"
	"sync"
)

var (
	chain *Blockchain
	once  sync.Once
)

func initChain(bc *Blockchain) {
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
