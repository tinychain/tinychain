package dpos_bft

import (
	"tinychain/common"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"sort"
	"sync"
)

type ProducersInfo struct {
	mu        sync.RWMutex
	all       map[peer.ID]*blockProducer
	bps       Producers // active block producers at current round
	currBP    int       // index of current block producer
	roundSize int       // max number of valid BP in one round
}

func newProducersInfo(maxSize int) *ProducersInfo {
	return &ProducersInfo{
		bps:       make([]*blockProducer, maxSize),
		roundSize: maxSize,
	}
}

func (pi *ProducersInfo) get(id peer.ID) *blockProducer {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.all[id]
}

func (pi *ProducersInfo) getAll() Producers {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	var bps Producers
	for _, bp := range pi.all {
		bps = append(bps, bp)
	}
	sort.Sort(bps)
	return bps
}

func (pi *ProducersInfo) next() {

}

func (pi *ProducersInfo) add(bp *blockProducer) {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	pi.all[bp.id] = bp
}

func (pi *ProducersInfo) remove(id peer.ID) {

}

func (pi *ProducersInfo) len() int {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return len(pi.bps)
}

type blockProducer struct {
	id      peer.ID        // bp peer's id
	address common.Address // address of bp to get reward
	privKey crypto.PrivKey // privKey
	pubKey  crypto.PubKey  // pubkey
	votes   uint64         // weight of vote
}

func (bp *blockProducer) Addr() common.Address {
	if bp.address.Nil() {
		if bp.pubKey != nil {
			addr, err := common.GenAddrByPubkey(bp.pubKey)
			if err != nil {
				return common.Address{}
			}
			bp.address = addr
		} else if bp.privKey != nil {
			addr, err := common.GenAddrByPrivkey(bp.privKey)
			if err != nil {
				return common.Address{}
			}
			bp.address = addr
		} else {
			return common.Address{}
		}
	}
	return bp.address
}

func (bp *blockProducer) PrivKey() crypto.PrivKey {
	return bp.privKey
}

type Producers []*blockProducer

func (p Producers) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Producers) Len() int {
	return len(p)
}

func (p Producers) Less(i, j int) bool {
	return p[i].votes > p[j].votes
}
