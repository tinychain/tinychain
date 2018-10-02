package pow

import (
	"encoding/json"
	"fmt"
	"github.com/libp2p/go-libp2p-crypto"
	"math"
	"math/big"
	"runtime"
	"sync/atomic"
	"time"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/consensus"
	"github.com/tinychain/tinychain/consensus/blockpool"
	"github.com/tinychain/tinychain/core"
	"github.com/tinychain/tinychain/core/state"
	"github.com/tinychain/tinychain/core/txpool"
	"github.com/tinychain/tinychain/core/types"
	"github.com/tinychain/tinychain/event"
	"github.com/tinychain/tinychain/p2p"
)

var (
	log = common.GetLogger("consensus")
)

const (
	maxNonce              = math.MaxUint64 << 5
	blockGapForDiffAdjust = 20160
	maxDiffculty          = math.MaxUint64 << 5
	minerReward           = 998
)

type consensusInfo struct {
	Difficulty uint64 `json:"Difficulty"` // difficulty target bits for mining
	Nonce      uint64 `json:"nonce"`      // computed result
}

func (ci *consensusInfo) Serialize() ([]byte, error) {
	return json.Marshal(ci)
}

// ProofOfWork implements proof-of-work consensus algorithm.
type ProofOfWork struct {
	config           *Config
	event            *event.TypeMux
	chain            consensus.Blockchain
	state            *state.StateDB
	blockPool        *blockpool.BlockPool
	txPool           *txpool.TxPool
	blValidator      consensus.BlockValidator // block validator
	csValidator      *csValidator             // consensus validator
	blockNum         uint64                   // new block num at certain difficulty period
	currMiningHeader *types.Header            // block header that being mined currently

	address common.Address
	privKey crypto.PrivKey

	quitCh      chan struct{}
	processLock chan struct{}
	mineStopCh  chan struct{} // listen for outer event to make miner stop

	// cache
	currDiff uint64 // current difficulty

	consensusSub event.Subscription // listen for the new proposed block executed by executor
	newTxsSub    event.Subscription // listen for the new pending transactions from txpool
	newBlockSub  event.Subscription // listen for the new block from block pool
	commitSub    event.Subscription // listen for the commit block completed from executor
	receiptsSub  event.Subscription // listen for the receipts executed by executor
	errSub       event.Subscription // listen for the executor's error event
}

func New(config *common.Config, state *state.StateDB, chain consensus.Blockchain, blValidator consensus.BlockValidator, txValidator consensus.TxValidator) (*ProofOfWork, error) {
	conf := newConfig(config)

	csValidator := newCsValidator(chain)

	pow := &ProofOfWork{
		config:      conf,
		chain:       chain,
		state:       state,
		blValidator: blValidator,
		csValidator: csValidator,
		event:       event.GetEventhub(),
		quitCh:      make(chan struct{}),
		blockPool:   blockpool.NewBlockPool(config, blValidator, csValidator, log, common.ProposeBlockMsg),
	}

	// if is miner node
	if conf.Miner {
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
	pow.commitSub = pow.event.Subscribe(&core.CommitCompleteEvent{})
	pow.receiptsSub = pow.event.Subscribe(&core.NewReceiptsEvent{})
	pow.newBlockSub = pow.event.Subscribe(&core.BlockReadyEvent{})
	pow.errSub = pow.event.Subscribe(&core.ErrOccurEvent{})

	go pow.listen()
	return nil
}

func (pow *ProofOfWork) listen() {
	for {
		select {
		case ev := <-pow.consensusSub.Chan():
			event := ev.(*core.ConsensusEvent)
			go pow.seal(event.Block, event.Receipts)
		case <-pow.newBlockSub.Chan():
			go pow.process()
		case ev := <-pow.receiptsSub.Chan():
			rev := ev.(*core.NewReceiptsEvent)
			go pow.validateAndCommit(rev.Block, rev.Receipts)
		case ev := <-pow.commitSub.Chan():
			block := ev.(*core.CommitCompleteEvent).Block
			go pow.commitComplete(block)
		case ev := <-pow.errSub.Chan():
			err := ev.(*core.ErrOccurEvent).Err
			log.Errorf("error occurs at executor process: %s", err)
			pow.processLock <- struct{}{}
			go pow.process()
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

func (pow *ProofOfWork) difficulty() (uint64, error) {
	if diff := atomic.LoadUint64(&pow.currDiff); diff != 0 {
		return diff, nil
	}
	last := pow.chain.LastBlock()
	consensus := consensusInfo{}
	err := json.Unmarshal(last.ConsensusInfo(), &consensus)
	if err != nil {
		return 0, err
	}
	atomic.StoreUint64(&pow.currDiff, consensus.Difficulty)
	return consensus.Difficulty, nil
}

// proposeBlock proposes a new pure block
func (pow *ProofOfWork) proposeBlock() {
	last := pow.chain.LastBlock()
	header := &types.Header{
		ParentHash: last.ParentHash(),
		Height:     last.Height(),
		Coinbase:   pow.address,
		Time:       new(big.Int).SetInt64(time.Now().Unix()),
		GasLimit:   pow.config.GasLimit,
	}

	block := types.NewBlock(header, pow.txPool.Pending())
	log.Infof("Block producer %s propose a new block, height = #%d", pow.Addr(), block.Height())
	go pow.event.Post(&core.ProposeBlockEvent{block})
}

func (pow *ProofOfWork) process() {
	block := pow.blockPool.GetBlock(pow.chain.LastBlock().Height() + 1)
	if block == nil {
		return
	}
	if pow.currMiningHeader != nil && pow.currMiningHeader.Height == block.Height() {
		if err := pow.csValidator.Validate(block); err == nil {
			pow.mineStopCh <- struct{}{}
		} else {
			return
		}
	}
	<-pow.processLock
	if block != nil {
		go pow.event.Post(&core.ExecBlockEvent{block})
	} else {
		pow.processLock <- struct{}{}
	}
}

// run performs proof-of-work.
// It will return error if other miner found a block with the same height
func (pow *ProofOfWork) run(block *types.Block) error {
	var (
		abort = make(chan struct{})
		found = make(chan uint64)
		n     = runtime.NumCPU()
		avg   = maxNonce / n
	)
	newDiff, err := pow.adjustDiff()
	if err != nil {
		return fmt.Errorf("cannot adjust difficulty when start to mine block #%d", block.Height())
	}
	pow.currMiningHeader = block.Header
	for i := 0; i < n; i++ {
		go newWorker(newDiff, uint64(avg*i), uint64(avg*(i+1)), block.Header).run(found, abort)
	}
	defer close(found)

	var nonce uint64
	select {
	case nonce = <-found:
		close(abort)
	case <-pow.mineStopCh:
		// close all mining work
		close(abort)
		pow.processLock <- struct{}{}
		return fmt.Errorf("stop mining, a block with the same height #%d found", block.Height())
	}

	csinfo := &consensusInfo{
		Difficulty: newDiff,
		Nonce:      nonce,
	}
	data, err := csinfo.Serialize()
	if err != nil {
		return fmt.Errorf("failed to encode consensus info, err:%s", err)
	}
	block.Header.ConsensusInfo = data
	pow.currMiningHeader = nil
	return nil
}

func (pow *ProofOfWork) Finalize(header *types.Header, state *state.StateDB, txs types.Transactions, receipts types.Receipts) (*types.Block, error) {
	// reward miner
	pow.state.AddBalance(header.Coinbase, new(big.Int).SetUint64(minerReward))

	// calculate state root
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

func (pow *ProofOfWork) seal(block *types.Block, receipts types.Receipts) error {
	if err := pow.run(block); err != nil {
		go pow.event.Post(&core.RollbackEvent{})
		return err
	}

	return pow.validateAndCommit(block, receipts)
}

// validateAndCommit validate block's state and consensus info, and finally commit it.
func (pow *ProofOfWork) validateAndCommit(block *types.Block, receipts types.Receipts) error {
	if err := pow.blValidator.ValidateState(block, pow.state, receipts); err != nil {
		log.Errorf("invalid block state, err:%s", err)
		go pow.event.Post(&core.RollbackEvent{})
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

func (pow *ProofOfWork) commitComplete(block *types.Block) {
	pow.blockPool.UpdateChainHeight(block.Height())
	pow.txPool.Drop(block.Transactions)

	pow.processLock <- struct{}{}
	go pow.broadcast(block)
	if !pow.config.Miner {
		go pow.process()
	} else {
		go pow.proposeBlock()
	}
}

func (pow *ProofOfWork) broadcast(block *types.Block) error {
	data, err := block.Serialize()
	if err != nil {
		return err
	}

	go pow.event.Post(&p2p.BroadcastEvent{
		Typ:  common.ProposeBlockMsg,
		Data: data,
	})
	return nil
}

func (pow *ProofOfWork) Protocols() []p2p.Protocol {
	return nil
}

// adjustDiff adjust difficulty of mining blocks
func (pow *ProofOfWork) adjustDiff() (uint64, error) {
	curr := pow.chain.LastBlock()
	if atomic.LoadUint64(&pow.blockNum) < blockGapForDiffAdjust {
		return pow.difficulty()
	}
	old := pow.chain.GetBlockByHeight(curr.Height() - blockGapForDiffAdjust)
	currDiff, err := pow.difficulty()
	if err != nil {
		return 0, err
	}
	newDiff := computeNewDiff(currDiff, curr, old)
	if newDiff > maxDiffculty {
		newDiff = maxDiffculty
	}
	atomic.StoreUint64(&pow.currDiff, newDiff)
	atomic.StoreUint64(&pow.blockNum, 0)
	return newDiff, nil
}

func (pow *ProofOfWork) rollback() {
	pow.event.Post(&core.RollbackEvent{})
}
