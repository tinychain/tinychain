package executor

import (
	"tinychain/core"
	"tinychain/event"
	"tinychain/core/state"
	"tinychain/core/types"
	batcher "github.com/yyh1102/go-batcher"
	"tinychain/common"
	"errors"
)

var (
	ErrBlockFallbehind = errors.New("block falls behind the current chain")

	log = common.GetLogger("executor")
)

// Processor represents the interface of block processor
type Processor interface {
	Process(block *types.Block) (types.Receipts, error)
}

type Executor struct {
	processor Processor
	chain     *core.Blockchain // Blockchain wrapper
	validator BlockValidator   // Block validator
	batch     batcher.Batch    // Batch for creating new block
	event     *event.TypeMux
	quitCh    chan struct{}

	execblockSub event.Subscription // Subscribe new block event
	execTxsSub   event.Subscription // Execute pending txs event
}

func New(config *Config, chain *core.Blockchain, statedb *state.StateDB) *Executor {
	processor := core.NewStateProcessor(chain, statedb)
	executor := &Executor{
		processor: processor,
		chain:     chain,
		validator: NewBlockValidator(config, chain),
		event:     event.GetEventhub(),
		quitCh:    make(chan struct{}),
	}
	return executor
}

func (ex *Executor) Start() error {
	ex.execblockSub = ex.event.Subscribe(&core.ExecBlockEvent{})
	ex.execTxsSub = ex.event.Subscribe(&core.ExecPendingTxEvent{})
	go ex.listen()
	return nil
}

func (ex *Executor) listen() {
	for {
		select {
		case ev := <-ex.execblockSub.Chan():
			block := ev.(*core.ExecBlockEvent).Block
			if err := ex.processBlock(block); err != nil {
				log.Errorf("failed to process block %s, err:%s", err)
			}
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
	// TODO execute block
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
