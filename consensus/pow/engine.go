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
)

var (
	log = common.GetLogger("consensus")
)

const (
	DEFAULT_TARGET_BITS = 24 // default mining difficulty
	MAX_NONCE           = math.MaxUint64
)

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
	event      *event.TypeMux
	difficulty uint64 // difficulty target bits

	address common.Address
	privKey crypto.PrivKey

	quitCh chan struct{}

	consensusSub event.Subscription
}

func New(config *common.Config) *ProofOfWork {
	// TODO config resolve
	return &ProofOfWork{
		event:  event.GetEventhub(),
		quitCh: make(chan struct{}),
	}
}

func (pow *ProofOfWork) Start() error {
	pow.consensusSub = pow.event.Subscribe(&core.ConsensusEvent{})
	return nil
}

func (pow *ProofOfWork) listen() {
	for {
		select {
		case ev := <-pow.consensusSub.Chan():
			block := ev.(*core.ConsensusEvent).Block
			go pow.Run(block)
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

func (pow *ProofOfWork) Run(block *types.Block) {
	n := runtime.NumCPU()
	avg := MAX_NONCE - n
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

func (pow *ProofOfWork) commit(block *types.Block) {
	go pow.event.Post(&core.CommitBlockEvent{block})
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
