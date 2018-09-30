package api

import (
	"tinychain/common"
	"tinychain/rpc/utils"
	"tinychain/tiny"
)

type ChainAPI struct {
	tiny *tiny.Tiny
}

func (api *ChainAPI) GetBlock(hash common.Hash, height uint64) *utils.Block {
	blk := api.tiny.Chain().GetBlock(hash, height)
	if blk == nil {
		return nil
	}
	return convertBlock(blk)
}

func (api *ChainAPI) GetBlockHash(height uint64) common.Hash {
	return api.tiny.Chain().GetHash(height)
}

func (api *ChainAPI) GetBlockByHash(hash common.Hash) *utils.Block {
	blk := api.tiny.Chain().GetBlockByHash(hash)
	return convertBlock(blk)
}

func (api *ChainAPI) GetBlockByHeight(height uint64) *utils.Block {
	blk := api.tiny.Chain().GetBlockByHeight(height)
	return convertBlock(blk)
}

func (api *ChainAPI) GetHeader(hash common.Hash, height uint64) *utils.Header {
	header := api.tiny.Chain().GetHeader(hash, height)
	return convertHeader(header)
}

func (api *ChainAPI) GetHeaderByHash(hash common.Hash) *utils.Header {
	height, err := api.tiny.DB().GetHeight(hash)
	if err != nil {
		return nil
	}
	return api.GetHeader(hash, height)
}

// Height returns the latest block height
func (api *ChainAPI) Height() uint64 {
	return api.tiny.Chain().LastHeight()
}

// Hash returns the latest block hash
func (api *ChainAPI) Hash() common.Hash {
	return api.tiny.Chain().LastBlock().Hash()
}

