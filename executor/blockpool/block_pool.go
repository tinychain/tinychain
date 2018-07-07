package blockpool

import (
	"sync"
	"tinychain/event"
	"tinychain/core/types"
	"tinychain/core"
	"math/big"
	"github.com/pkg/errors"
	"tinychain/common"
	batcher "github.com/yyh1102/go-batcher"
	"sort"
)

var (
	log = common.GetLogger("blockpool")

	ErrBlockDuplicate  = errors.New("block duplicate")
	ErrPoolFull        = errors.New("block pool is full")
	ErrBlockFallbehind = errors.New("block falls behind the current chain")
)

type BlockValidator interface {
	ValidateHeader(block *types.Block) error
	ValidateBody(block *types.Block) error
}

type Blockchain interface {
	LastBlock() *types.Block
}

type BlockPool struct {
	config    *Config
	mu        sync.RWMutex
	chain     Blockchain                // current blockchain
	validator BlockValidator            // Block validator
	valid     map[*big.Int]*types.Block // Valid blocks pool. map[height]*block
	batch     *batcher.Batch            // Batch for blocks launching
	event     *event.TypeMux
	quitCh    chan struct{}

	blockSub  event.Subscription
	commitSub event.Subscription // Receive msg when new blocks are appended to blockchain
}

func NewBlockPool(config *Config, validator BlockValidator) *BlockPool {
	bp := &BlockPool{
		config:    config,
		event:     event.GetEventhub(),
		validator: validator,
		valid:     make(map[*big.Int]*types.Block, config.MaxBlockSize),
	}

	batch := batcher.NewBatch(
		"APPEND_VALID_BLOCK",
		config.BatchCapacity,
		config.BatchTimeout,
		bp.launch,
	)

	bp.batch = batch
	return bp
}

func (bp *BlockPool) Start() {

	bp.blockSub = bp.event.Subscribe(&core.NewBlockEvent{})
	bp.commitSub = bp.event.Subscribe(&core.BlockCommitEvent{})
	go bp.listen()
}

func (bp *BlockPool) listen() {
	for {
		select {
		case ev := <-bp.blockSub.Chan():
			block := ev.(*core.NewBlockEvent).Block
			go bp.add(block)
		case ev := <-bp.commitSub.Chan():
			commit := ev.(*core.BlockCommitEvent)
			go bp.delBlocks(commit.Heights)
		case <-bp.quitCh:
			bp.blockSub.Unsubscribe()
			return
		}
	}
}

// launch implements cbFunc in batcher.
// It will be invoked and post a batch of valid blocks when reaches batch size or timeout.
func (bp *BlockPool) launch(batch []interface{}) {
	var blocks types.Blocks
	for _, item := range batch {
		blocks = append(blocks, item.(*types.Block))
	}
	// sort blocks by height-asec-order
	sort.Sort(blocks)
	appendBlockEv := &core.AppendBlockEvent{
		Blocks: blocks,
	}
	go bp.event.Post(appendBlockEv)
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

	// Validate block
	if err := bp.validate(block); err != nil {
		return err
	}

	bp.valid[block.Height()] = block
	return nil
}

func (bp *BlockPool) validate(block *types.Block) error {
	if block.Height().Cmp(bp.chain.LastBlock().Height()) < 0 {
		return ErrBlockFallbehind
	}
	err := bp.validator.ValidateHeader(block)
	if err != nil {
		log.Errorf("Error occurs when validating block header whose height is %s, %s", block.Height(), err)
		return err
	}

	err = bp.validator.ValidateBody(block)
	if err != nil {
		log.Errorf("Error occurs when validating block body whose height is %s, %s", block.Height(), err)
		return err
	}
	return nil
}

// delBlocks remove the blocks with given height.
func (bp *BlockPool) delBlocks(heights []*big.Int) {
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
