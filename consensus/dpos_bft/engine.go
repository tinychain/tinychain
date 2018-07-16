package dpos_bft

import (
	"tinychain/core/types"
	"tinychain/core/state"
	"tinychain/event"
	"tinychain/core"
	"tinychain/common"
	"time"
	"math/big"
	"sync/atomic"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
)

var (
	log = common.GetLogger("consensus")
)

const (
	// Period type of the BFT algorithm
	PROPOSE    = iota
	PRE_COMMIT
	COMMIT
)

type Blockchain interface {
	LastBlock() *types.Block
}

type Engine struct {
	config   *Config
	chain    Blockchain // current blockchain
	peers    *peerPool
	bp       *blockProducer
	bftState atomic.Value
	event    *event.TypeMux
	quitCh   chan struct{}

	newTxsSub    event.Subscription // Subscribe new txs event from tx_pool
	consensusSub event.Subscription // Subscribe kick_off bft event
}

func NewEngine(config *common.Config, chain Blockchain, id peer.ID) (*Engine, error) {
	conf := newConfig(config)
	privKey, err := crypto.UnmarshalPrivateKey(conf.PrivKey)
	if err != nil {
		return nil, err
	}
	// init block producer info
	bp, err := newBP(conf, id, privKey)
	if err != nil {
		log.Errorf("failed to initialize the info of block producer")
		return nil, err
	}
	return &Engine{
		config: conf,
		chain:  chain,
		event:  event.GetEventhub(),
		bp:     bp,
		quitCh: make(chan struct{}),
		peers:  newPeerPool(bp),
	}, nil
}

func (eg *Engine) Start() error {
	eg.newTxsSub = eg.event.Subscribe(&core.NewTxsEvent{})
	eg.consensusSub = eg.event.Subscribe(&core.ConsensusEvent{})

	go eg.listen()
	return nil
}

func (eg *Engine) Stop() error {
	return nil
}

func (eg *Engine) listen() {
	for {
		select {
		case ev := <-eg.newTxsSub.Chan():
			txs := ev.(*core.NewTxsEvent).Txs
			go eg.proposeBlock(txs, nil)
		case <-eg.quitCh:
			eg.newTxsSub.Unsubscribe()
			return
		}
	}
}

func (eg *Engine) Address() common.Address {
	return eg.bp.address
}

func (eg *Engine) getState() int {
	if s := eg.bftState.Load(); s != nil {
		return s.(int)
	}
	return 0
}

func (eg *Engine) setState(priod int) {
	eg.bftState.Store(priod)
}

// proposeBlock proposes a new block without state_root and receipts_root
func (eg *Engine) proposeBlock(txs types.Transactions, extra []byte) {
	header := &types.Header{
		ParentHash: eg.chain.LastBlock().Hash(),
		Height:     eg.chain.LastBlock().Height() + 1,
		Coinbase:   eg.Address(),
		Extra:      extra,
		Time:       new(big.Int).SetInt64(time.Now().Unix()),
		GasLimit:   eg.config.GasLimit,
	}

	block := types.NewBlock(header, txs)
	go eg.event.Post(&core.ProposeBlockEvent{block})
}

func (eg *Engine) Finalize(header *types.Header, state *state.StateDB, txs types.Transactions, receipts types.Receipts) (*types.Block, error) {
	root, err := state.IntermediateRoot()
	if err != nil {
		return nil, err
	}
	header.StateRoot = root
	header.ReceiptsHash = receipts.Hash()

	header.TxRoot = txs.Hash()
	return types.NewBlock(header, txs), nil
}
