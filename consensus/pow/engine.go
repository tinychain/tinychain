package pow

import (
	"tinychain/common"
	"math"
	"runtime"
	"tinychain/core/types"
	"tinychain/event"
	"tinychain/core"
	"encoding/json"
	"sync"
	"tinychain/core/state"
	"github.com/libp2p/go-libp2p-crypto"
	"tinychain/p2p"
	"math/big"
	"time"
	"tinychain/consensus/blockpool"
	"tinychain/executor"
	"tinychain/core/txpool"
)

var (
	log = common.GetLogger("consensus")
)

const (
	DEFAULT_TARGET_BITS = 24 // default mining difficulty
	MAX_NONCE           = math.MaxUint64
)

type Blockchain interface {
	LastBlock() *types.Block
}

type consensusInfo struct {
	Difficulty uint64 `json:"Difficulty"` // difficulty target bits for mining
	Nonce      uint64 `json:"nonce"`      // computed result
}

func (ci *consensusInfo) Serialize() ([]byte, error) {
	return json.Marshal(ci)
}

func (ci *consensusInfo) Deserialize(d []byte) error {
	return json.Unmarshal(d, ci)
}

// ProofOfWork implements proof-of-work consensus algorithm.
type ProofOfWork struct {
	config      *Config
	event       *event.TypeMux
	chain       Blockchain
	state       *state.StateDB
	blockPool   common.BlockPool
	txPool      *txpool.TxPool
	blValidator executor.BlockValidator // block validator
	csValidator *csValidator            // consensus validator
	difficulty  uint64                  // difficulty target bits

	address common.Address
	privKey crypto.PrivKey

	quitCh      chan struct{}
	processLock chan struct{}

	consensusSub event.Subscription // listen for the new proposed block executed by executor
	newTxsSub    event.Subscription // listen for the new pending transactions from txpool
	newBlockSub  event.Subscription // listen for the new block from block pool
	commitSub    event.Subscription // listen for the commit block completed from executor
	receiptsSub  event.Subscription // listen for the receipts executed by executor
}

func New(config *common.Config, chain Blockchain, state *state.StateDB, validator executor.BlockValidator) (*ProofOfWork, error) {
	// TODO config resolve
	conf := newConfig(config)

	csValidator := newCsValidator(conf.Difficulty)

	pow := &ProofOfWork{
		config:      conf,
		chain:       chain,
		state:       state,
		blValidator: validator,
		csValidator: csValidator,
		event:       event.GetEventhub(),
		quitCh:      make(chan struct{}),
		blockPool:   blockpool.NewBlockPool(config, validator, csValidator, log, common.PROPOSE_BLOCK_MSG),
	}

	txValidator := executor.NewTxValidator(executor.NewConfig(config), state)
	// if is mining node
	if conf.Mining {
		privKey, err := crypto.UnmarshalPrivateKey(conf.PrivKey)
		if err != nil {
			return nil, err
		}
		pow.privKey = privKey
		pow.address, err = common.GenAddrByPrivkey(privKey)
		if err != nil {
			return nil, err
		}
		pow.txPool = txpool.NewTxPool(config, txValidator, state, true, false)
	} else {
		pow.txPool = txpool.NewTxPool(config, txValidator, state, true, true)
	}

	return pow, nil
}

func (pow *ProofOfWork) Start() error {
	pow.consensusSub = pow.event.Subscribe(&core.ConsensusEvent{})
	pow.newTxsSub = pow.event.Subscribe(&core.ExecPendingTxEvent{})
	pow.commitSub = pow.event.Subscribe(&core.CommitCompleteEvent{})
	pow.receiptsSub = pow.event.Subscribe(&core.NewReceiptsEvent{})
	pow.newBlockSub = pow.event.Subscribe(&core.BlockReadyEvent{})
	return nil
}

func (pow *ProofOfWork) listen() {
	for {
		select {
		case ev := <-pow.consensusSub.Chan():
			block := ev.(*core.ConsensusEvent).Block
			go pow.run(block)
		case ev := <-pow.newTxsSub.Chan():
			txs := ev.(*core.ExecPendingTxEvent).Txs
			go pow.proposeBlock(txs)
		case <-pow.newBlockSub.Chan():
			go pow.process()
		case ev := <-pow.receiptsSub.Chan():
			rev := ev.(*core.NewReceiptsEvent)
			go pow.validateAndCommit(rev.Block, rev.Receipts)
		case ev := <-pow.commitSub.Chan():
			block := ev.(*core.CommitCompleteEvent).Block
			pow.blockPool.UpdateChainHeight(pow.chain.LastBlock().Height())

			pow.processLock <- struct{}{}
			go pow.broadcast(block)
			if !pow.config.Mining {
				go pow.process()
			}
		case <-pow.quitCh:
			pow.consensusSub.Unsubscribe()
			return
		}
	}
}

func (pow *ProofOfWork) Stop() error {
	close(pow.quitCh)
	return nil
}

func (pow *ProofOfWork) Addr() common.Address {
	return pow.address
}

// proposeBlock proposes a new pure block
func (pow *ProofOfWork) proposeBlock(txs types.Transactions) {
	last := pow.chain.LastBlock()
	header := &types.Header{
		ParentHash: last.ParentHash(),
		Height:     last.Height(),
		Coinbase:   pow.address,
		Time:       new(big.Int).SetInt64(time.Now().Unix()),
		GasLimit:   pow.config.GasLimit,
	}

	block := types.NewBlock(header, txs)
	go pow.event.Post(&core.ProposeBlockEvent{block})
	log.Infof("Block producer %s propose a new block, height = #%d", pow.Addr(), block.Height())
}

func (pow *ProofOfWork) process() {
	<-pow.processLock
	block := pow.blockPool.GetBlock(pow.chain.LastBlock().Height() + 1)
	if block != nil {
		go pow.event.Post(&core.ExecBlockEvent{block})
	} else {
		pow.processLock <- struct{}{}
	}
}

func (pow *ProofOfWork) run(block *types.Block) {
	n := runtime.NumCPU()
	avg := MAX_NONCE / n
	foundChan := newNonceChan()
	for i := 0; i < n; i++ {
		go newWorker(pow.difficulty, uint64(avg*i), uint64(avg*(i+1)), block.Clone()).Run(foundChan)
	}

	nonce := <-foundChan.ch
	// close channel and notify other workers to stop mining
	foundChan.close()

	consensus := &consensusInfo{
		Difficulty: pow.difficulty,
		Nonce:      nonce,
	}
	data, err := consensus.Serialize()
	if err != nil {
		log.Errorf("failed to encode consensus info, err:%s", err)
		return
	}
	block.Header.ConsensusInfo = data
	pow.commit(block)
}

func (pow *ProofOfWork) Finalize(header *types.Header, state *state.StateDB, txs types.Transactions, receipts types.Receipts) (*types.Block, error) {
	root, err := state.IntermediateRoot()
	if err != nil {
		return nil, err
	}
	header.StateRoot = root
	header.ReceiptsHash = receipts.Hash()

	header.TxRoot = txs.Hash()
	newBlk := types.NewBlock(header, txs)
	newBlk.PubKey, err = pow.privKey.GetPublic().Bytes()
	if err != nil {
		return nil, err
	}
	newBlk.Sign(pow.privKey)
	return newBlk, nil
}

// validateAndCommit validate block's state and consensus info, and finally commit it.
func (pow *ProofOfWork) validateAndCommit(block *types.Block, receipts types.Receipts) error {
	if err := pow.blValidator.ValidateState(block, pow.state, receipts); err != nil {
		log.Errorf("invalid block state, err:%s", err)
		return err
	}

	if err := pow.csValidator.Validate(block); err != nil {
		log.Errorf("invalid block consensus info, err:%s", err)
		return err
	}
	pow.commit(block)
	return nil
}

func (pow *ProofOfWork) commit(block *types.Block) {
	go pow.event.Post(&core.CommitBlockEvent{block})
}

func (pow *ProofOfWork) broadcast(block *types.Block) error {
	data, err := block.Serialize()
	if err != nil {
		return err
	}

	go pow.event.Post(&p2p.BroadcastEvent{
		Typ:  common.PROPOSE_BLOCK_MSG,
		Data: data,
	})
	return nil
}

func (pow *ProofOfWork) Protocols() []p2p.Protocol {
	return nil
}

type nonceChan struct {
	mu     sync.RWMutex
	closed bool
	ch     chan uint64
}

func newNonceChan() *nonceChan {
	return &nonceChan{
		ch: make(chan uint64, 1),
	}
}

func (nc *nonceChan) post(nonce uint64) {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	if !nc.closed {
		nc.ch <- nonce
	}
}

func (nc *nonceChan) close() {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	if !nc.closed {
		nc.closed = true
		close(nc.ch)
	}
}
