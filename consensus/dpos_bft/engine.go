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
	"tinychain/p2p"
	"github.com/op/go-logging"
)

const (
	// Period type of the BFT algorithm
	PROPOSE    = iota
	PRE_COMMIT
	COMMIT

	BLOCK_PROPOSE_GAP = 5 * time.Second
)

type BlockPool interface {
	p2p.Protocol
	GetBlock(height uint64) *types.Block
	Clear(height uint64)
}

type Blockchain interface {
	LastBlock() *types.Block
}

// Engine is the main wrapper of dpos_bft algorithm
// The dpos_bft algorithm process procedures described below:
// 1. select the 21 block producer at a round, and shuffle the order of them.
// 2. every selected block producers propose a new block in turn, and multicast to other BPs.
// 3. Wait 0.5s for network io, and this BP kick off the bft process.
// 4. After bft performs successfully, all BPs commit the blocks.
type Engine struct {
	config *Config
	log    *logging.Logger

	seqNo    atomic.Value // current block height at bft processing
	chain    Blockchain   // current blockchain
	bps      *bpsMgr      // manage and operate block producers' info
	bpool    BlockPool    // pool to retrieves new proposed blocks
	bftState atomic.Value // the state of current bft period

	preCommitVotes int // pre-commit votes statistics
	commitVotes    int // commit votes statistics

	event  *event.TypeMux
	quitCh chan struct{}

	newTxsSub    event.Subscription // Subscribe new txs event from tx_pool
	consensusSub event.Subscription // Subscribe kick_off bft event
}

func New(config *common.Config, log *logging.Logger, chain Blockchain, id peer.ID, bpool BlockPool) (*Engine, error) {
	conf := newConfig(config)
	privKey, err := crypto.UnmarshalPrivateKey(conf.PrivKey)
	if err != nil {
		return nil, err
	}
	// init block producer info
	self := &blockProducer{
		id:      id,
		privKey: privKey,
	}
	return &Engine{
		config: conf,
		chain:  chain,
		bps:    newBPsMgr(conf, log, self, chain),
		event:  event.GetEventhub(),
		bpool:  bpool,
		quitCh: make(chan struct{}),
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

// proposeLoop set a loop to try to propose a block every 2s
func (eg *Engine) proposeLoop() {
	ticker := time.NewTicker(BLOCK_PROPOSE_GAP)
	for {
		select {
		case <-ticker.C:
			self := eg.bps.reachSelfTurn()
			if self == nil {
				continue
			}
			// TODO try to check the current state of bps and decide to propose block or not
			// 1. if the rate of vote is lower than 15%, select bp randomly
			// 2. if the rate of votes is higher than 15%, select the highest 21 bps to produce blocks in turn

		}
	}
}

func (eg *Engine) Address() common.Address {
	return eg.bps.self.Addr()
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

func (eg *Engine) Protocols() []p2p.Protocol {
	return []p2p.Protocol{
		eg,
		eg.bps,
		eg.bpool,
	}
}
