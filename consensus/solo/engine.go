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
	"tinychain/core/txpool"
	"tinychain/executor"
	"tinychain/p2p"
)

var (
	log = common.GetLogger("consensus")
)

type Blockchain interface {
	LastBlock() *types.Block
}

type TxPool interface {
}

type SoloEngine struct {
	config    *Config
	chain     Blockchain
	blockPool consensus.BlockPool
	txPool    TxPool
	state     *state.StateDB
	event     *event.TypeMux

	processLock chan struct{} // channel lock to prevent concurrent block processing

	privKey crypto.PrivKey

	newBlockSub  event.Subscription // listen for the new block from
	newTxsSub    event.Subscription // listen for the new pending transactions from txpool
	commitSub    event.Subscription // listen for the commit block completed from executor
	consensusSub event.Subscription // listen for the new proposed block after executing
}

func NewSoloEngine(config *common.Config, state *state.StateDB, chain Blockchain) (*SoloEngine, error) {
	conf := newConfig(config)
	privKey, err := crypto.UnmarshalPrivateKey(conf.PrivKey)
	if err != nil {
		return nil, err
	}
	if conf.Address.Nil() {
		conf.Address, err = common.GenAddrByPrivkey(privKey)
		if err != nil {
			return nil, err
		}
	}
	return &SoloEngine{
		config:    conf,
		privKey:   privKey,
		event:     event.GetEventhub(),
		blockPool: blockpool.NewBlockPool(config, log, common.PROPOSE_BLOCK_MSG),
		txPool:    txpool.NewTxPool(config, executor.NewTxValidator(executor.NewConfig(config), state), state, true),
	}, nil
}

func (solo *SoloEngine) Start() error {
	solo.newBlockSub = solo.event.Subscribe(&core.BlockReadyEvent{})
	solo.commitSub = solo.event.Subscribe(&core.CommitCompleteEvent{})
	solo.newTxsSub = solo.event.Subscribe(&core.ExecPendingTxEvent{})
	solo.consensusSub = solo.event.Subscribe(&core.ConsensusEvent{})

	solo.processLock <- struct{}{}

	go solo.listen()
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
			go solo.randomCast(block)
			go solo.process()
		case ev := <-solo.commitSub.Chan():
			block := ev.(*core.CommitCompleteEvent).Block
			solo.processLock <- struct{}{}
			go solo.broadcast(block)
			go solo.process() // call next process
		case ev := <-solo.consensusSub.Chan():
			block := ev.(*core.ConsensusEvent).Block
			go solo.commit(block)
		}
	}
}

func (solo *SoloEngine) process() {
	<-solo.processLock
	block := solo.blockPool.GetBlock(solo.chain.LastBlock().Height() + 1)
	if block != nil {
		go solo.event.Post(&core.ExecBlockEvent{block})
	} else {
		solo.processLock <- struct{}{}
	}
}

func (solo *SoloEngine) Address() common.Address {
	return solo.config.Address
}

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

func (solo *SoloEngine) randomCast(block *types.Block) error {
	data, err := block.Serialize()
	if err != nil {
		return err
	}
	go solo.event.Post(&p2p.RandomSendEvnet{
		Typ:  common.PROPOSE_BLOCK_MSG,
		Data: data,
	})
	return nil
}

func (solo *SoloEngine) commit(block *types.Block) {
	go solo.event.Post(&core.CommitBlockEvent{block})
}
