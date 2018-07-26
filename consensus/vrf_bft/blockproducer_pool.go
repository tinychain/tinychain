package vrf_bft

import (
	"tinychain/p2p/pb"
	msg "tinychain/consensus/vrf_bft/message"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/golang/protobuf/proto"
	"tinychain/event"
	"tinychain/p2p"
	"tinychain/common"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/op/go-logging"
	"sync/atomic"
)

const (
	NEW_BP_REG  = iota
	OLD_BP_RESP
)

type Peer interface {
	ID() peer.ID
}

// bpPool manage and operate block producers' state at a certain consensus round
type bpPool struct {
	log *logging.Logger

	dposActive bool           // check votes rate, use random selects defaultly
	chain      Blockchain     // current blockchain
	bpsInfo    *ProducersInfo // block producers state
	currInd    int            // index of current block producer
	bpsCache   atomic.Value   // selected producers cache

	event *event.TypeMux
	self  *blockProducer
}

func newBpPool(config *Config, log *logging.Logger, bp *blockProducer, chain Blockchain) *bpPool {
	return &bpPool{
		log:     log,
		self:    bp,
		chain:   chain,
		bpsInfo: newProducersInfo(config.RoundSize),
		event:   event.GetEventhub(),
	}
}

func (bm *bpPool) get(id peer.ID) *blockProducer {
	return bm.bpsInfo.get(id)
}

// getBPs returns the block producers at current round
func (bm *bpPool) getBPs() Producers {
	if bps := bm.bpsCache.Load(); bps != nil {
		return bps.(Producers)
	}
	return bm.selectBPs()
}

// selectBPs selects the block producers set of this round according to a given rule.
// The default rule is determined as below:
// 1. if the rate of vote is lower than 15%, select bp randomly
// 2. if the rate of votes is higher than 15%, select the highest 21 bps to produce blocks in turn
func (bm *bpPool) selectBPs() Producers {
	var bps Producers
	bm.currInd = 0
	if bm.dposActive {
		bps = bm.bpsInfo.getDposBPs()
	} else {
		bps = bm.bpsInfo.getRandomBPs(bm.chain.LastBlock().Hash())
	}
	bm.bpsCache.Store(bps)
	return bps
}

// reachSelfTurn checks is it the turn for self bp.
// It will return bp obj if it's its turn.
func (bm *bpPool) reachSelfTurn() *blockProducer {
	producers := bm.bpsCache.Load()
	if producers == nil {
		bm.selectBPs()
	}
	bps := producers.(Producers)
	if bps[bm.currInd].Cmp(bm.self) {
		return bm.self
	}
	bm.currInd++
	if bm.currInd > len(bps) {
		bm.selectBPs()
	}
	return nil
}

func (bm *bpPool) add(id peer.ID, pubKey crypto.PubKey) {
	bm.bpsInfo.add(&blockProducer{
		id:     id,
		pubKey: pubKey,
	})
}

func (bm *bpPool) count() int {
	return bm.bpsInfo.len()
}

func (bm *bpPool) Type() string {
	return common.CONSENSUS_PEER_MSG
}

func (bm *bpPool) Run(pid peer.ID, message *pb.Message) error {
	peerMsg := msg.PeerMsg{}
	err := proto.Unmarshal(message.Data, &peerMsg)
	if err != nil {
		bm.log.Errorf("failed to unmarshal message from p2p, err:%s", err)
		return err
	}

	pubkey, err := crypto.UnmarshalPublicKey(peerMsg.PubKey)
	if err != nil {
		bm.log.Errorf("failed to unserialize pubkey from %s, err:%s", err, peerMsg.Type)
	}

	bm.add(pid, pubkey)

	// If is new bp, response self peer id and public key
	if peerMsg.Type == NEW_BP_REG {
		pubkeyBytes, err := bm.self.pubKey.Bytes()
		if err != nil {
			bm.log.Errorf("failed serialize pubkey as bytes, err:%s", err)
			return err
		}

		data, err := proto.Marshal(&msg.PeerMsg{
			Type:   OLD_BP_RESP,
			Id:     []byte(peer.ID(bm.self.id)),
			PubKey: pubkeyBytes,
		})
		if err != nil {
			bm.log.Errorf("failed to marshal peer message, err:%s", err)
			return err
		}
		go bm.event.Post(&p2p.SendMsgEvent{
			Target: peer.ID(peerMsg.Id),
			Typ:    common.CONSENSUS_PEER_MSG,
			Data:   data,
		})
	}
	return nil
}

func (bm *bpPool) Error(err error) {
	bm.log.Errorf("bpPool error: %s", err)
}
