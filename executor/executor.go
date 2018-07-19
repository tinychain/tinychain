package executor

import (
	"tinychain/core"
	"tinychain/event"
	"tinychain/core/state"
	"tinychain/core/types"
	batcher "github.com/yyh1102/go-batcher"
	"tinychain/common"
	"errors"
	"tinychain/consensus"
	bpool "tinychain/common/blockpool"
	"sync/atomic"
	"tinychain/db"
	"sync"
)

var (
	ErrBlockFallbehind = errors.New("block falls behind the current chain")

	log = common.GetLogger("executor")
)

type BlockPool interface {
	GetBlock(height uint64) *types.Block
	Clear(height uint64)
}

// Processor represents the interface of block processor
type Processor interface {
	Process(block *types.Block) (types.Receipts, error)
}

type Blockchain interface {
	LastBlock() *types.Block
	AddBlock(block *types.Block) error
	CommitBlock(block *types.Block) error
}

type Executor struct {
	db        *db.TinyDB
	processor Processor
	chain     Blockchain     // Blockchain wrapper
	validator BlockValidator // Block validator
	batch     batcher.Batch  // Batch for creating new block
	blockpool BlockPool      // Block pool for new block caching
	state     *state.StateDB
	engine    consensus.Engine
	event     *event.TypeMux
	quitCh    chan struct{}

	receiptsCache sync.Map // receipts cache, map[uint64]types.Receipts

	processing atomic.Value // Processing state, 1 means processing, 0 means idle

	blockReadySub   event.Subscription // Subscribe new block ready event from block_pool
	proposeBlockSub event.Subscription // Subscribe propose new block event
	commitSub       event.Subscription // Subscribe state commit event
}

func New(config *common.Config, db *db.TinyDB, chain *core.Blockchain, statedb *state.StateDB, engine consensus.Engine) *Executor {
	processor := core.NewStateProcessor(chain, statedb, engine)
	executor := &Executor{
		db:        db,
		processor: processor,
		chain:     chain,
		validator: NewBlockValidator(chain),
		engine:    engine,
		event:     event.GetEventhub(),
		quitCh:    make(chan struct{}),
		blockpool: bpool.NewBlockPool(config, log, common.READY_BLOCK_MSG),
	}
	return executor
}

func (ex *Executor) Start() error {
	ex.blockReadySub = ex.event.Subscribe(&core.ExecBlockEvent{})
	ex.proposeBlockSub = ex.event.Subscribe(&core.ProposeBlockEvent{})
	ex.commitSub = ex.event.Subscribe(&core.CommitBlock{})

	go ex.listen()
	return nil
}

func (ex *Executor) listen() {
	for {
		select {
		case ev := <-ex.blockReadySub.Chan():
			height := ev.(*core.BlockReadyEvent).Height
			if height == ex.lastHeight()+1 {
				go ex.process()
			}
		case ev := <-ex.proposeBlockSub.Chan():
			block := ev.(*core.ProposeBlockEvent).Block
			go ex.proposeBlock(block)
		case ev := <-ex.commitSub.Chan():
			block := ev.(*core.CommitBlock).Block
			go ex.commit(block)
		case <-ex.quitCh:
			ex.proposeBlockSub.Unsubscribe()
			ex.commitSub.Unsubscribe()
			ex.blockReadySub.Unsubscribe()
			return
		}
	}
}

func (ex *Executor) Stop() error {
	close(ex.quitCh)
	return nil
}

func (ex *Executor) lastHeight() uint64 {
	return ex.chain.LastBlock().Height()
}

// processState get the current processing state, and returns 1 processing, or 0 idle
func (ex *Executor) processState() int {
	if p := ex.processing.Load(); p != nil {
		return p.(int)
	}
	return 0
}

// process set a infinite loop to process block in the order of height.
func (ex *Executor) process() error {
	isProcessing := ex.processState()
	if isProcessing == 1 {
		return nil
	}
	ex.processing.Store(1)
	defer ex.processing.Store(0)
	for {
		nextBlk := ex.blockpool.GetBlock(ex.lastHeight() + 1)
		if nextBlk == nil {
			break
		}
		if err := ex.processBlock(nextBlk); err != nil {
			// TODO Roll back, and drop the future blocks
			return err
		}
	}
	ex.blockpool.Clear(ex.lastHeight())
	return nil
}

// processBlock process the validation and execution of a received block from other peers
func (ex *Executor) processBlock(block *types.Block) error {

	if block.Height() < ex.chain.LastBlock().Height() {
		return ErrBlockFallbehind
	}
	if err := ex.validator.ValidateHeader(block); err != nil {
		log.Errorf("failed to validate block #%d header, err:%s", block.Height(), err)
		return err
	}

	receipts, err := ex.execBlock(block)
	if err != nil {
		log.Errorf("failed to execute block #%d, err:%s", block.Height(), err)
		return err
	}

	if err := ex.validator.ValidateBody(block, receipts); err != nil {
		log.Errorf("failed to validate block #%d body, err:%s", block.Height(), err)
		return err
	}

	// Save receipts to cache
	ex.receiptsCache.Store(block.Height(), receipts)

	if err := ex.chain.AddBlock(block); err != nil {
		log.Errorf("failed to add block to blockchain cache, err:%s", err)
		return err
	}

	return nil
}

// proposeBlock executes new transactions from tx_pool and pack a new block.
// The new block is created by consensus engine and does not include state_root, tx_root and receipts_root.
func (ex *Executor) proposeBlock(block *types.Block) error {
	receipts, err := ex.execBlock(block)
	if err != nil {
		log.Errorf("failed to exec block #%d, err:%s", block.Height(), err)
		return err
	}

	// Save receipts to cache
	ex.receiptsCache.Store(block.Height(), receipts)

	if _, err := ex.engine.Finalize(block.Header, ex.state, block.Transactions, receipts); err != nil {
		log.Errorf("failed to finalize the block #%d, err:%s", block.Height(), err)
		return err
	}

	go ex.event.Post(&core.ConsensusEvent{block})
	return nil
}

// execBlock process block in state
func (ex *Executor) execBlock(block *types.Block) (types.Receipts, error) {
	return ex.processor.Process(block)
}
