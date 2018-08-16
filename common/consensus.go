package common

import (
	"tinychain/p2p"
	"tinychain/core/types"
)

type BlockPool interface {
	p2p.Protocol
	GetBlock(height uint64) *types.Block
	AddBlock(block *types.Block) error
	DelBlock(block *types.Block)
	UpdateChainHeight(height uint64)
	Clear(height uint64)
}
