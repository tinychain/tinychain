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
	bp "tinychain/executor/blockpool"
	"sync/atomic"
)

var (
	ErrBlockFallbehind = errors.New("block falls behind the current chain")

	log = common.GetLogger("executor")
)

type BlockPool interface {
	GetBlock(height uint64) *types.Block
	DelBlock(height uint64) *types.Block
}

// Processor represents the interface of block processor
type Processor interface {
	Process(block *types.Block) (types.Receipts, error)
}

type Blockchain interface {
	LastBlock() *types.Block
	AddBlock(block *types.Block) error
}

type Executor struct {
	processor Processor
	chain     Blockchain     // Blockchain wrapper
	validator BlockValidator // Block validator
	batch     batcher.Batch  // Batch for creating new block
	blockpool BlockPool      // Block pool for new block caching
	engine    consensus.Engine
	event     *event.TypeMux
	quitCh    chan struct{}

	processing atomic.Value // Processing state, 1 means processing, 0 means idle

	blockReadySub event.Subscription // Subscribe new block ready event
	execTxsSub    event.Subscription // Execute pending txs event
}

func New(config *common.Config, chain *core.Blockchain, statedb *state.StateDB, engine consensus.Engine) *Executor {
	processor := core.NewStateProcessor(chain, statedb, engine)
	executor := &Executor{
		processor: processor,
		chain:     chain,
		validator: NewBlockValidator(config, chain),
		engine:    engine,
		event:     event.GetEventhub(),
		quitCh:    make(chan struct{}),
		blockpool: bp.NewBlockPool(config),
	}
	return executor
}

func (ex *Executor) Start() error {
	ex.blockReadySub = ex.event.Subscribe(&core.ExecBlockEvent{})
	ex.execTxsSub = ex.event.Subscribe(&core.ExecPendingTxEvent{})
	go ex.listen()
	return nil
}

func (ex *Executor) listen() {
	for {
		select {
		case <-ex.blockReadySub.Chan():
			go ex.process()
		case ev := <-ex.execTxsSub.Chan():
			txs := ev.(*core.ExecPendingTxEvent).Txs
			go ex.processTx(txs)
		case <-ex.quitCh:
			ex.execTxsSub.Unsubscribe()
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

func (ex *Executor) process() error {
	if isProcessing := ex.processing.Load(); isProcessing != nil {
		if isProcessing.(int) == 1 {
			return nil
		}
	}
	ex.processing.Store(1)
	for {
		nextHeight := ex.lastHeight() + 1
		nextBlk := ex.blockpool.GetBlock(nextHeight)
		if nextBlk == nil {
			break
		}
		ex.processBlock(nextBlk)
	}
	ex.processing.Store(0)
}

func (ex *Executor) processBlock(block *types.Block) error {
	if block.Height() < ex.chain.LastBlock().Height() {
		return ErrBlockFallbehind
	}
	if err := ex.validator.ValidateHeader(block); err != nil {
		log.Errorf("error occurs when validating block #%d header, err:%s", block.Height(), err)
		return err
	}

	receipts, err := ex.execBlock(block)
	if err != nil {
		log.Errorf("error occurs when executing block #%d, err:%s", block.Height(), err)
	}

	if err := ex.validator.ValidateBody(block, receipts); err != nil {
		log.Errorf("error occurs when validating block #%d body, err:%s", block.Height(), err)
		return err
	}

	if err := ex.chain.AddBlock(block); err != nil {

	}
	return nil
}

func (ex *Executor) execBlock(block *types.Block) (types.Receipts, error) {
	return ex.processor.Process(block)
}

func (ex *Executor) validateBlock() error {

}

func (ex *Executor) genNewBlock(txs types.Transactions, receipts types.Receipts) (*types.Block, error) {

}

// processTx execute transactions launched from tx_pool.
// 1. Simulate execute every transaction sequentially, until gasUsed reaches blocks's gasLimit
// 2. Collect valid txs and invalid txs
// 3. Collect receipts (remove invalid receipts)
func (ex *Executor) processTx(txs types.Transactions) {
}
