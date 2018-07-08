package blockpool

import (
	"sync"
	"tinychain/event"
	"tinychain/core/types"
	"tinychain/core"
	"errors"
	"tinychain/common"
	"time"
)

var (
	log = common.GetLogger("blockpool")

	ErrBlockDuplicate  = errors.New("block duplicate")
	ErrPoolFull        = errors.New("block pool is full")
)

type Blockchain interface {
	LastBlock() *types.Block
	AddBlocks(blocks types.Blocks) error
}

type BlockPool struct {
	config    *Config
	mu        sync.RWMutex
	chain     Blockchain              // Current blockchain
	valid     map[uint64]*types.Block // Valid blocks pool. map[height]*block
	nextBlkCh chan *types.Block       // Next block to process
	event     *event.TypeMux
	quitCh    chan struct{}

	blockSub event.Subscription
	commitSub event.Subscription
}

func NewBlockPool(config *Config) *BlockPool {
	bp := &BlockPool{
		config:    config,
		event:     event.GetEventhub(),
		valid:     make(map[uint64]*types.Block, config.MaxBlockSize),
		nextBlkCh: make(chan *types.Block, config.MaxBlockSize),
		quitCh:    make(chan struct{}),
	}

	return bp
}

func (bp *BlockPool) Start() {
	bp.blockSub = bp.event.Subscribe(&core.NewBlockEvent{})
	bp.commitSub=bp.event.Subscribe(&core.BlockCommitEvent{})

	go bp.listen()
	go bp.watch()
	go bp.launch()
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

// launch
func (bp *BlockPool) launch() {
	for {
		select {
		case next := <-bp.nextBlkCh:
			go bp.event.Post(&core.ExecBlockEvent{
				Block: next,
			})
		case <-bp.quitCh:
			return
		}
	}
}

// watch watches the block valid pool, and post next block to process
func (bp *BlockPool) watch() {
	timer := time.NewTicker(bp.config.WatchInterval)
	for {
		select {
		case <-timer.C:
			lastHeight := bp.chain.LastBlock().Height()
			bp.mu.Lock()
			if next, ok := bp.valid[lastHeight+1]; ok {
				bp.nextBlkCh <- next
			}
			bp.mu.Unlock()
		case <-bp.quitCh:
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
	if bp.Size() >= bp.config.MaxBlockSize && old == nil {
		return ErrPoolFull
	}

	bp.valid[block.Height()] = block
	return nil
}

// delBlocks remove the blocks with given height.
func (bp *BlockPool) delBlocks(heights []uint64) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	for _, height := range heights {
		delete(bp.valid, height)
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
