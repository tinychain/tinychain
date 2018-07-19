package blockpool

import (
	"sync"
	"tinychain/event"
	"tinychain/core/types"
	"tinychain/core"
	"errors"
	"tinychain/common"
	"tinychain/p2p/pb"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/op/go-logging"
	"encoding/json"
)

var (
	ErrBlockDuplicate = errors.New("block duplicate")
	ErrPoolFull       = errors.New("block pool is full")
)

type BlockPool struct {
	maxBlockSize uint64
	mu           sync.RWMutex
	log          *logging.Logger
	msgType      string                  // Message type for p2p transfering
	valid        map[uint64]*types.Block // Valid blocks pool. map[height]*block
	event        *event.TypeMux
	quitCh       chan struct{}

	blockSub event.Subscription // Subscribe new block received from p2p
}

func NewBlockPool(config *common.Config, log *logging.Logger, msgType string) *BlockPool {
	maxBlockSize := uint64(config.GetInt64(common.MAX_BLOCK_SIZE))
	bp := &BlockPool{
		maxBlockSize: maxBlockSize,
		event:        event.GetEventhub(),
		log:          log,
		msgType:      msgType,
		valid:        make(map[uint64]*types.Block, maxBlockSize),
		quitCh:       make(chan struct{}),
	}

	return bp
}

//func (bp *BlockPool) Start() {
//	bp.blockSub = bp.event.Subscribe(&core.NewBlockEvent{})
//
//	go bp.listen()
//}
//
//func (bp *BlockPool) listen() {
//	for {
//		select {
//		case ev := <-bp.blockSub.Chan():
//			block := ev.(*core.NewBlockEvent).Block
//			go bp.add(block)
//		case <-bp.quitCh:
//			bp.blockSub.Unsubscribe()
//			return
//		}
//	}
//}

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
func (bp *BlockPool) Size() uint64 {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	return uint64(len(bp.valid))
}

//func (bp *BlockPool) Stop() {
//	close(bp.quitCh)
//}

func (bp *BlockPool) Type() string {
	return bp.msgType
}

func (bp *BlockPool) Run(pid peer.ID, message *pb.Message) error {
	block := types.Block{}
	err := json.Unmarshal(message.Data, &block)
	if err != nil {
		return err
	}
	bp.add(&block)
	return nil
}

func (bp *BlockPool) Error(err error) {
	bp.log.Errorf("blockpool error: %s", err)
}
