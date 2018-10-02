package dpos_bft

import (
	"github.com/tinychain/tinychain/p2p/pb"
	msg "github.com/tinychain/tinychain/consensus/dpos_bft/message"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/golang/protobuf/proto"
	"github.com/tinychain/tinychain/event"
	"github.com/tinychain/tinychain/p2p"
	"github.com/tinychain/tinychain/common"
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

// peerPool manage and operate block producers' state at a certain consensus round
type peerPool struct {
	log *logging.Logger

	dposActive bool           // check votes rate, use random selects defaultly
	chain      Blockchain     // current blockchain
	bpsInfo    *ProducersInfo // block producers state
	currInd    int            // index of current block producer
	bpsCache   atomic.Value   // selected producers cache

	event *event.TypeMux
	self  *blockProducer
}

func newBpPool(config *Config, log *logging.Logger, bp *blockProducer, chain Blockchain) *peerPool {
	return &peerPool{
		log:     log,
		self:    bp,
		chain:   chain,
		bpsInfo: newProducersInfo(config.RoundSize),
		event:   event.GetEventhub(),
	}
}

func (pl *peerPool) get(id peer.ID) *blockProducer {
	return pl.bpsInfo.get(id)
}

// getBPs returns the block producers at current round
func (pl *peerPool) getBPs() Producers {
	if bps := pl.bpsCache.Load(); bps != nil {
		return bps.(Producers)
	}
	return pl.selectBPs()
}

// selectBPs selects the block producers set of this round according to a given rule.
// The default rule is determined as below:
// 1. if the rate of vote is lower than 15%, select bp randomly
// 2. if the rate of votes is higher than 15%, select the highest 21 bps to produce blocks in turn
func (pl *peerPool) selectBPs() Producers {
	var bps Producers
	pl.currInd = 0
	if pl.dposActive {
		bps = pl.bpsInfo.getDposBPs()
	} else {
		bps = pl.bpsInfo.getRandomBPs(pl.chain.LastBlock().Hash())
	}
	pl.bpsCache.Store(bps)
	return bps
}

// reachSelfTurn checks is it the turn for self bp.
// It will return bp obj if it's its turn.
func (pl *peerPool) reachSelfTurn() *blockProducer {
	producers := pl.bpsCache.Load()
	if producers == nil {
		pl.selectBPs()
	}
	bps := producers.(Producers)
	if bps[pl.currInd].Cmp(pl.self) {
		return pl.self
	}
	pl.currInd++
	if pl.currInd > len(bps) {
		pl.selectBPs()
	}
	return nil
}

func (pl *peerPool) add(id peer.ID, pubKey crypto.PubKey) {
	pl.bpsInfo.add(&blockProducer{
		id:     id,
		pubKey: pubKey,
	})
}

func (pl *peerPool) count() int {
	return pl.bpsInfo.len()
}

func (pl *peerPool) Type() string {
	return common.ConsensusPeerMsg
}

func (pl *peerPool) Run(pid peer.ID, message *pb.Message) error {
	peerMsg := msg.PeerMsg{}
	err := proto.Unmarshal(message.Data, &peerMsg)
	if err != nil {
		pl.log.Errorf("failed to unmarshal message from p2p, err:%s", err)
		return err
	}

	pubkey, err := crypto.UnmarshalPublicKey(peerMsg.PubKey)
	if err != nil {
		pl.log.Errorf("failed to unserialize pubkey from %s, err:%s", err, peerMsg.Type)
	}

	pl.add(pid, pubkey)

	// If is new bp, response self peer id and public key
	if peerMsg.Type == NEW_BP_REG {
		pubkeyBytes, err := pl.self.pubKey.Bytes()
		if err != nil {
			pl.log.Errorf("failed serialize pubkey as bytes, err:%s", err)
			return err
		}

		data, err := proto.Marshal(&msg.PeerMsg{
			Type:   OLD_BP_RESP,
			Id:     []byte(peer.ID(pl.self.id)),
			PubKey: pubkeyBytes,
		})
		if err != nil {
			pl.log.Errorf("failed to marshal peer message, err:%s", err)
			return err
		}
		go pl.event.Post(&p2p.SendMsgEvent{
			Target: peer.ID(peerMsg.Id),
			Typ:    common.ConsensusPeerMsg,
			Data:   data,
		})
	}
	return nil
}

func (pl *peerPool) Error(err error) {
	pl.log.Errorf("peerPool error: %s", err)
}
