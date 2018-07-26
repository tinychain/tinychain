package vrf_bft

import (
	"tinychain/common"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
	"sort"
	"sync"
	"math/rand"
	"bytes"
)

type ProducersInfo struct {
	mu        sync.RWMutex
	all       map[peer.ID]*blockProducer
	roundSize int // max number of valid BP in one round
}

func newProducersInfo(roundSize int) *ProducersInfo {
	return &ProducersInfo{
		all:       make(map[peer.ID]*blockProducer),
		roundSize: roundSize,
	}
}

func (pi *ProducersInfo) get(id peer.ID) *blockProducer {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.all[id]
}

func (pi *ProducersInfo) getAll() Producers {
	pi.mu.RLock()
	var bps Producers
	for _, bp := range pi.all {
		bps = append(bps, bp)
	}
	pi.mu.RUnlock()
	sort.Sort(bps)
	return bps
}

// getDposBPs get the given number of bps at the current round according to DPOS mechanism
func (pi *ProducersInfo) getDposBPs() Producers {
	return pi.getAll()[:pi.roundSize]
}

// getRandomBPs get the random list of bps. The random seed is according to the prev_block_hash
func (pi *ProducersInfo) getRandomBPs(prevHash common.Hash) Producers {
	bps := pi.getAll()
	rand.Shuffle(len(bps), func(i, j int) {
		bps[i], bps[j] = bps[j], bps[i]
	})
	return bps[:pi.roundSize]
}

func (pi *ProducersInfo) next() {

}

func (pi *ProducersInfo) add(bp *blockProducer) {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	pi.all[bp.id] = bp
}

func (pi *ProducersInfo) remove(id peer.ID) {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	delete(pi.all, id)
}

func (pi *ProducersInfo) len() int {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return len(pi.all)
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

func (bp *blockProducer) PubKey() crypto.PubKey {
	return bp.pubKey
}

func (bp *blockProducer) PrivKey() crypto.PrivKey {
	return bp.privKey
}

func (bp *blockProducer) Cmp(p *blockProducer) bool {
	pubKey1, err := bp.pubKey.Bytes()
	if err != nil {
		panic("invalid public key")
	}
	pubKey2, err := p.pubKey.Bytes()
	if err != nil {
		panic("invalid public key")
	}
	return bytes.Compare(pubKey1, pubKey2) == 0 && bp.id == p.id
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
