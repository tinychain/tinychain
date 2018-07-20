package dpos_bft

import (
	"tinychain/p2p/pb"
	msg "tinychain/consensus/dpos_bft/message"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/golang/protobuf/proto"
	"tinychain/event"
	"tinychain/p2p"
	"tinychain/common"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/op/go-logging"
)

const (
	NEW_BP_REG  = iota
	OLD_BP_RESP
)

type Peer interface {
	ID() peer.ID
}

// bpsMgr manage and operate block producers' state at a certain consensus round
type bpsMgr struct {
	log *logging.Logger

	chain    Blockchain
	bpsInfo  *ProducersInfo // block producers state
	currInd  int            // index of current block producer
	bpsCache Producers      // selected producers cache

	event *event.TypeMux
	self  *blockProducer
}

func newBPsMgr(config *Config, log *logging.Logger, bp *blockProducer, chain Blockchain) *bpsMgr {
	return &bpsMgr{
		log:     log,
		self:    bp,
		chain:   chain,
		bpsInfo: newProducersInfo(config.RoundSize),
		event:   event.GetEventhub(),
	}
}

func (bm *bpsMgr) get(id peer.ID) *blockProducer {
	return bm.bpsInfo.get(id)
}

func (bm *bpsMgr) selectBPs(dpos bool) {
	bm.currInd = 0
	if dpos {
		bm.bpsCache = bm.bpsInfo.getDposBPs()
	} else {
		bm.bpsCache = bm.bpsInfo.getRandomBPs(bm.chain.LastBlock().Hash())
	}
}

// reachSelfTurn checks is it the turn for self bp.
// It will return bp obj if it's its turn.
func (bm *bpsMgr) reachSelfTurn() *blockProducer {
	if bm.bpsCache[bm.currInd] == bm.self {
		return bm.self
	}
	bm.currInd++
	if bm.currInd > len(bm.bpsCache) {
		// TODO check votes rate, use random selects defaultly
		bm.selectBPs(false)
	}
	return nil
}

func (bm *bpsMgr) add(id peer.ID, pubKey crypto.PubKey) {
	bm.bpsInfo.add(&blockProducer{
		id:     id,
		pubKey: pubKey,
	})
}

func (bm *bpsMgr) count() int {
	return bm.bpsInfo.len()
}

func (bm *bpsMgr) Type() string {
	return common.CONSENSUS_PEER_MSG
}

func (bm *bpsMgr) Run(pid peer.ID, message *pb.Message) error {
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

func (bm *bpsMgr) Error(err error) {
	bm.log.Errorf("bpsMgr error: %s", err)
}
