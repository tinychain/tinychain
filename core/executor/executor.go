package executor

import (
	"fmt"
	batcher "github.com/yyh1102/go-batcher"
	"sync"
	"sync/atomic"
	"tinychain/common"
	"tinychain/consensus"
	"tinychain/core"
	"tinychain/core/chain"
	"tinychain/core/state"
	"tinychain/core/types"
	"tinychain/db"
	"tinychain/event"
)

var (
	log = common.GetLogger("executor")
)

// Processor represents the interface of block processor
type Processor interface {
	Process(block *types.Block) (types.Receipts, error)
}

type Executor struct {
	conf      *common.Config
	db        *db.TinyDB
	processor Processor
	chain     *chain.Blockchain // Blockchain wrapper
	batch     batcher.Batch     // Batch for creating new block
	state     *state.StateDB
	engine    consensus.Engine
	event     *event.TypeMux
	quitCh    chan struct{}

	receiptsCache sync.Map // receipts cache, map[uint64]types.Receipts

	processing atomic.Value // Processing state, 1 means processing, 0 means idle

	execBlockSub    event.Subscription // Subscribe new block ready event from block_pool
	proposeBlockSub event.Subscription // Subscribe propose new block event
	commitSub       event.Subscription // Subscribe state commit event
}

func New(config *common.Config, db *db.TinyDB, chain *chain.Blockchain, engine consensus.Engine) *Executor {
	executor := &Executor{
		conf:   config,
		db:     db,
		chain:  chain,
		engine: engine,
		event:  event.GetEventhub(),
		quitCh: make(chan struct{}),
	}
	return executor
}

func (ex *Executor) Init() error {
	genesis := ex.chain.Genesis()
	if genesis == nil {
		newGenesis, err := ex.createGenesis()
		if err != nil {
			log.Errorf("failed to create genesis when init executor, %s", err)
			return err
		}
		genesis = newGenesis
	}
	statedb, err := state.New(ex.db.LDB(), genesis.StateRoot().Bytes())
	if err != nil {
		log.Errorf("failed to init state when init executor, %s", err)
		return err
	}
	ex.state = statedb
	ex.processor = NewStateProcessor(ex.conf, ex.chain, statedb, ex.engine)
	return nil
}

func (ex *Executor) Start() error {
	ex.execBlockSub = ex.event.Subscribe(&core.ExecBlockEvent{})
	ex.proposeBlockSub = ex.event.Subscribe(&core.ProposeBlockEvent{})
	ex.commitSub = ex.event.Subscribe(&core.CommitBlockEvent{})

	go ex.listen()
	return nil
}

func (ex *Executor) listen() {
	for {
		select {
		case ev := <-ex.proposeBlockSub.Chan():
			block := ev.(*core.ProposeBlockEvent).Block
			go ex.proposeBlock(block)
		case ev := <-ex.execBlockSub.Chan():
			block := ev.(*core.ExecBlockEvent).Block
			go ex.applyBlock(block)
		case ev := <-ex.commitSub.Chan():
			block := ev.(*core.CommitBlockEvent).Block
			go ex.commit(block)
		case <-ex.quitCh:
			ex.proposeBlockSub.Unsubscribe()
			ex.commitSub.Unsubscribe()
			ex.execBlockSub.Unsubscribe()
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

//// process set a infinite loop to process block in the order of height.
//func (ex *Executor) process() error {
//	isProcessing := ex.processState()
//	if isProcessing == 1 {
//		return nil
//	}
//	ex.processing.Store(1)
//	defer ex.processing.Store(0)
//	for {
//		nextBlk := ex.engine.BlockPool().GetBlock(ex.lastHeight() + 1)
//		if nextBlk == nil {
//			break
//		}
//		if err := ex.processBlock(nextBlk); err != nil {
//			// TODO Roll back, and drop the future blocks
//			return err
//		}
//	}
//	return nil
//}

// applyBlockBlock process the validation and execute the received block.
func (ex *Executor) applyBlock(block *types.Block) error {
	if currHeight := ex.chain.LastBlock().Height(); block.Height() != currHeight+1 {
		return fmt.Errorf("block height is not match, demand #%d, got #%d", currHeight+1, block.Height())
	}
	ex.state.UpdateCurrHeight(block.Height())
	receipts, err := ex.execBlock(block)
	if err != nil {
		log.Errorf("failed to execute block #%d, err:%s", block.Height(), err)
		return err
	}

	// Save receipts to cache
	ex.receiptsCache.Store(block.Height(), receipts)

	if err := ex.chain.AddBlock(block); err != nil {
		log.Errorf("failed to add block to blockchain cache, err:%s", err)
		return err
	}

	// Send receipts to engine
	go ex.event.Post(&core.NewReceiptsEvent{
		Block:    block,
		Receipts: receipts,
	})

	return nil
}

// proposeBlock executes new transactions from tx_pool and pack a new block.
// The new block is created by consensus engine and does not include state_root, tx_root and receipts_root.
func (ex *Executor) proposeBlock(block *types.Block) error {
	ex.state.UpdateCurrHeight(block.Height())
	receipts, err := ex.execBlock(block)
	if err != nil {
		log.Errorf("failed to exec block #%d, err:%s", block.Height(), err)
		return err
	}

	// Save receipts to cache
	ex.receiptsCache.Store(block.Height(), receipts)

	newBlk, err := ex.engine.Finalize(block.Header, ex.state, block.Transactions, receipts)
	if err != nil {
		log.Errorf("failed to finalize the block #%d, err:%s", block.Height(), err)
		return err
	}

	go ex.event.Post(&core.ConsensusEvent{newBlk})
	return nil
}

// execBlock process block in state
func (ex *Executor) execBlock(block *types.Block) (types.Receipts, error) {
	return ex.processor.Process(block)
}
