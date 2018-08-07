package solo

import (
	"tinychain/core/types"
	"tinychain/consensus"
	"tinychain/common"
	"tinychain/core/state"
	"github.com/libp2p/go-libp2p-crypto"
	"tinychain/consensus/blockpool"
	"tinychain/event"
	"tinychain/core"
	"tinychain/p2p"
	"sync/atomic"
	"tinychain/executor"
	"tinychain/core/txpool"
)

var (
	log = common.GetLogger("consensus")
)

const (
	BP  = iota // block producer
	NBP        // not block producer
)

type Blockchain interface {
	LastBlock() *types.Block
}

type TxPool interface {
	Start()
	Stop()
}

type SoloEngine struct {
	typ       atomic.Value
	config    *Config
	chain     Blockchain
	validator consensus.BlockValidator
	blockPool consensus.BlockPool
	txPool    TxPool
	state     *state.StateDB
	event     *event.TypeMux

	processLock chan struct{} // channel lock to prevent concurrent block processing
	quitCh      chan struct{}

	address common.Address
	privKey crypto.PrivKey

	newBlockSub  event.Subscription // listen for the new block from block pool
	newTxsSub    event.Subscription // listen for the new pending transactions from txpool
	consensusSub event.Subscription // listen for the new proposed block from executor
	receiptsSub  event.Subscription // listen for the receipts after executing
	commitSub    event.Subscription // listen for the commit block completed from executor
}

func NewSoloEngine(config *common.Config, state *state.StateDB, chain Blockchain, validator consensus.BlockValidator) (*SoloEngine, error) {
	conf := newConfig(config)
	soloEngine := &SoloEngine{
		config:    conf,
		event:     event.GetEventhub(),
		blockPool: blockpool.NewBlockPool(config, validator, log, common.PROPOSE_BLOCK_MSG),
	}
	if conf.BP {
		privKey, err := crypto.UnmarshalPrivateKey(conf.PrivKey)
		if err != nil {
			return nil, err
		}
		soloEngine.privKey = privKey
		soloEngine.address, err = common.GenAddrByPrivkey(privKey)
		if err != nil {
			return nil, err
		}
		soloEngine.typ.Store(BP)
		soloEngine.txPool = txpool.NewTxPool(config, executor.NewTxValidator(executor.NewConfig(config), state), state, true, false)
	} else {
		soloEngine.typ.Store(NBP)
		soloEngine.txPool = txpool.NewTxPool(config, executor.NewTxValidator(executor.NewConfig(config), state), state, true, true)
	}
	return soloEngine, nil
}

func (solo *SoloEngine) Start() error {
	solo.newBlockSub = solo.event.Subscribe(&core.BlockReadyEvent{})
	solo.commitSub = solo.event.Subscribe(&core.CommitCompleteEvent{})
	solo.newTxsSub = solo.event.Subscribe(&core.ExecPendingTxEvent{})
	solo.consensusSub = solo.event.Subscribe(&core.ConsensusEvent{})
	solo.receiptsSub = solo.event.Subscribe(&core.NewReceiptsEvent{})

	solo.processLock <- struct{}{}

	go solo.listen()
	go solo.txPool.Start()
	return nil
}

func (solo *SoloEngine) listen() {
	for {
		select {
		case ev := <-solo.newTxsSub.Chan():
			txs := ev.(*core.ExecPendingTxEvent).Txs
			go solo.proposeBlock(txs)
		case ev := <-solo.newBlockSub.Chan():
			block := ev.(*core.BlockReadyEvent).Block
			go solo.broadcast(block)
			go solo.process()
		case ev := <-solo.commitSub.Chan():
			block := ev.(*core.CommitCompleteEvent).Block
			solo.processLock <- struct{}{}
			go solo.broadcast(block)
			// if this peer is a NBP
			if solo.Type() == NBP {
				go solo.process() // call next process
			}
		case ev := <-solo.consensusSub.Chan():
			// this channel will be passed when executor completes to propose block,
			// and always done by bp
			block := ev.(*core.ConsensusEvent).Block
			go solo.commit(block)
		case ev := <-solo.receiptsSub.Chan():
			// this channel will be passed when executor completes to process block and ge receipts,
			// and always done by not-bp
			rev := ev.(*core.NewReceiptsEvent)
			go solo.validateAndCommit(rev.Block, rev.Receipts)
		case <-solo.quitCh:
			solo.newTxsSub.Unsubscribe()
			solo.newBlockSub.Unsubscribe()
			solo.commitSub.Unsubscribe()
			solo.consensusSub.Unsubscribe()
			solo.receiptsSub.Unsubscribe()
			return
		}
	}
}

func (solo *SoloEngine) Stop() error {
	solo.quitCh <- struct{}{}
	solo.txPool.Stop()
	return nil
}

func (solo *SoloEngine) Type() int {
	if val := solo.typ.Load(); val != nil {
		return val.(int)
	}
	return NBP
}

func (solo *SoloEngine) Address() common.Address {
	return solo.address
}

// process processes the block received from the solo BP.
// It is invoked by not-BP peers.
func (solo *SoloEngine) process() {
	<-solo.processLock
	block := solo.blockPool.GetBlock(solo.chain.LastBlock().Height() + 1)
	if block != nil {
		go solo.event.Post(&core.ExecBlockEvent{block})
	} else {
		solo.processLock <- struct{}{}
	}
}

// proposeBlock proposes a new block with given transactions retrieved from tx_pool.
func (solo *SoloEngine) proposeBlock(txs types.Transactions) {
	<-solo.processLock
	header := &types.Header{
		ParentHash: solo.chain.LastBlock().Hash(),
		Height:     solo.chain.LastBlock().Height() + 1,
		Coinbase:   solo.Address(),
		Extra:      solo.config.Extra,
	}

	block := types.NewBlock(header, txs)
	go solo.event.Post(&core.ProposeBlockEvent{block})
	log.Infof("Block producer %s propose a new block, height = #%d", solo.Address(), block.Height())
	solo.processLock <- struct{}{}
}

func (solo *SoloEngine) broadcast(block *types.Block) error {
	data, err := block.Serialize()
	if err != nil {
		return err
	}
	go solo.event.Post(&p2p.BroadcastEvent{
		Typ:  common.PROPOSE_BLOCK_MSG,
		Data: data,
	})
	return nil
}

func (solo *SoloEngine) Finalize(header *types.Header, state *state.StateDB, txs types.Transactions, receipts types.Receipts) (*types.Block, error) {
	root, err := state.IntermediateRoot()
	if err != nil {
		return nil, err
	}
	header.StateRoot = root
	header.ReceiptsHash = receipts.Hash()

	header.TxRoot = txs.Hash()
	newBlk := types.NewBlock(header, txs)
	newBlk.PubKey, err = solo.privKey.GetPublic().Bytes()
	if err != nil {
		return nil, err
	}
	newBlk.Sign(solo.privKey)
	return newBlk, nil
}

func (solo *SoloEngine) validateAndCommit(block *types.Block, receipts types.Receipts) error {
	if err := solo.validator.ValidateState(block, solo.state, receipts); err != nil {
		log.Errorf("invalid block state, err:%s", err)
		return err
	}
	solo.commit(block)
	return nil
}

func (solo *SoloEngine) commit(block *types.Block) {
	go solo.event.Post(&core.CommitBlockEvent{block})
}

func (solo *SoloEngine) Protocols() []p2p.Protocol {
	return nil
}
