package blockpool

import (
	"encoding/json"
	"errors"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/op/go-logging"
	"sync"
	"sync/atomic"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/consensus"
	"github.com/tinychain/tinychain/core"
	"github.com/tinychain/tinychain/core/types"
	"github.com/tinychain/tinychain/event"
	"github.com/tinychain/tinychain/p2p/pb"
)

var (
	ErrBlockDuplicate  = errors.New("block duplicate")
	ErrPoolFull        = errors.New("block pool is full")
	ErrBlockFallBehind = errors.New("invalid block: block height falls behind the currnet chain")
)

// ConsensusValidator validates the consensus info field in block header
type ConsensusValidator interface {
	Validate(block *types.Block) error
}

type BlockPool struct {
	maxBlockSize uint64
	mu           sync.RWMutex
	log          *logging.Logger
	msgType      string                  // Message type for p2p transfering
	valid        map[uint64]types.Blocks // Valid blocks pool. map[height]*block
	blValidator  consensus.BlockValidator
	csValidator  ConsensusValidator
	chainHeight  atomic.Value

	event  *event.TypeMux
	quitCh chan struct{}

	blockSub event.Subscription // Subscribe new block received from p2p
}

// Create a block pool instance
// The arg `msgType` tells the block pool to listen for the specified type of message from p2p layer
func NewBlockPool(config *common.Config, blValidator consensus.BlockValidator, csValidator ConsensusValidator, log *logging.Logger, msgType string) *BlockPool {
	maxBlockSize := uint64(config.GetInt64(common.MAX_BLOCK_NUM))
	bp := &BlockPool{
		maxBlockSize: maxBlockSize,
		event:        event.GetEventhub(),
		log:          log,
		msgType:      msgType,
		blValidator:  blValidator,
		csValidator:  csValidator,
		valid:        make(map[uint64]types.Blocks, maxBlockSize),
		quitCh:       make(chan struct{}),
	}

	return bp
}

// MsgType returns the msg type used in p2p layer
func (bp *BlockPool) MsgType() string {
	return bp.msgType
}

func (bp *BlockPool) GetBlock(height uint64) *types.Block {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	return bp.valid[height][0]
}

func (bp *BlockPool) GetBlocks(height uint64) types.Blocks {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	return bp.valid[height]
}

func (bp *BlockPool) isExist(block *types.Block) bool {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	blks := bp.valid[block.Height()]
	for _, blk := range blks {
		if blk.Hash() == block.Hash() {
			return true
		}
	}
	return false
}

// AddBlocks add a new block to pool without validating and processing it
func (bp *BlockPool) AddBlock(block *types.Block) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	if bp.isExist(block) {
		return ErrBlockDuplicate
	}
	bp.append(block)
	return nil
}

func (bp *BlockPool) add(block *types.Block) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	if block.Height() <= bp.ChainHeight() {
		return ErrBlockFallBehind
	}

	// Validate block
	if err := bp.blValidator.ValidateHeader(block.Header); err != nil {
		return err
	}

	if bp.csValidator != nil {
		if err := bp.csValidator.Validate(block); err != nil {
			return err
		}
	}
	// Check block duplicate
	if bp.isExist(block) {
		return ErrBlockDuplicate
	}
	if bp.Size() >= bp.maxBlockSize {
		// clear old blocks
		bp.Clear(bp.ChainHeight() - bp.maxBlockSize)
		if bp.Size() >= bp.maxBlockSize {
			return ErrPoolFull
		}
	}

	bp.append(block)
	go bp.event.Post(&core.BlockReadyEvent{block})
	return nil
}

// append push a new block to the given slot.
// This func requires the caller to hold write-lock.
func (bp *BlockPool) append(block *types.Block) {
	bp.valid[block.Height()] = append(bp.valid[block.Height()], block)
}

// DelBlock removes the block with given height.
func (bp *BlockPool) DelBlock(block *types.Block) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	blks := bp.valid[block.Height()]
	for i, blk := range blks {
		if blk.Hash() == block.Hash() {
			blks = append(blks[0:i], blks[i+1:]...)
		}
	}
	bp.valid[block.Height()] = blks
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

func (bp *BlockPool) ChainHeight() uint64 {
	if h := bp.chainHeight.Load(); h != nil {
		return h.(uint64)
	}
	return 0
}

func (bp *BlockPool) UpdateChainHeight(height uint64) {
	bp.chainHeight.Store(height)
	//bp.Clear(height)
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
