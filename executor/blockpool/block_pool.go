package blockpool

import (
	"sync"
	"tinychain/event"
	"tinychain/core/types"
	"tinychain/core"
	"errors"
	"tinychain/common"
)

var (
	log = common.GetLogger("blockpool")

	ErrBlockDuplicate = errors.New("block duplicate")
	ErrPoolFull       = errors.New("block pool is full")
)

type BlockPool struct {
	maxBlockSize uint64
	mu           sync.RWMutex
	valid        map[uint64]*types.Block // Valid blocks pool. map[height]*block
	event        *event.TypeMux
	quitCh       chan struct{}

	blockSub event.Subscription // Subscribe new block received from p2p
}

func NewBlockPool(config *common.Config) *BlockPool {
	maxBlockSize := uint64(config.GetInt64(common.MAX_BLOCK_SIZE))
	bp := &BlockPool{
		maxBlockSize: maxBlockSize,
		event:        event.GetEventhub(),
		valid:        make(map[uint64]*types.Block, maxBlockSize),
		quitCh:       make(chan struct{}),
	}

	return bp
}

func (bp *BlockPool) Start() {
	bp.blockSub = bp.event.Subscribe(&core.NewBlockEvent{})

	go bp.listen()
}

func (bp *BlockPool) listen() {
	for {
		select {
		case ev := <-bp.blockSub.Chan():
			block := ev.(*core.NewBlockEvent).Block
			go bp.add(block)
		case <-bp.quitCh:
			bp.blockSub.Unsubscribe()
			return
		}
	}
}

func (bp *BlockPool) Valid() []*types.Block {
	var blocks []*types.Block
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	for _, block := range bp.valid {
		blocks = append(blocks, block)
	}
	return blocks
}

func (bp *BlockPool) GetBlock(height uint64) *types.Block {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	return bp.valid[height]
}

func (bp *BlockPool) add(block *types.Block) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	// Check block duplicate
	old := bp.valid[block.Height()]
	if old != nil {
		// Replace the old block if new one has a higher gasused
		if old.Hash() != block.Hash() && old.GasUsed() > block.GasUsed() {
			return ErrBlockDuplicate
		}
	}
	if bp.Size() >= bp.maxBlockSize && old == nil {
		return ErrPoolFull
	}

	bp.valid[block.Height()] = block
	go bp.event.Post(&core.BlockReadyEvent{block.Height()})
	return nil
}

// DelBlock removes the block with given height.
func (bp *BlockPool) DelBlock(height uint64) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	delete(bp.valid, height)
}

// ClearByHeight removes the block whose height is lower than the given height
func (bp *BlockPool) Clear(height uint64) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	for h := range bp.valid {
		if height >= h {
			delete(bp.valid, h)
		}
	}
}

// Size gets the size of valid blocks.
// The caller should hold the lock before invoke this func.
func (bp *BlockPool) Size() uint64 {
	return uint64(len(bp.valid))
}

func (bp *BlockPool) Stop() {
	close(bp.quitCh)
}
