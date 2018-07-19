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

// bpsPool manage and operate block producers' state at a certain consensus round
type bpsPool struct {
	log     *logging.Logger
	bpsInfo *ProducersInfo // block producers state
	event   *event.TypeMux
	self    *blockProducer
}

func newBPsPool(config *Config, log *logging.Logger, bp *blockProducer) *bpsPool {
	return &bpsPool{
		log:     log,
		self:    bp,
		bpsInfo: newProducersInfo(config.RoundSize),
		event:   event.GetEventhub(),
	}
}

func (p *bpsPool) get(id peer.ID) *blockProducer {
	return p.bpsInfo.get(id)
}

func (p *bpsPool) add(id peer.ID, pubKey crypto.PubKey) {
	p.bpsInfo.add(&blockProducer{
		id:     id,
		pubKey: pubKey,
	})
}

func (p *bpsPool) count() int {
	return p.bpsInfo.len()
}

func (p *bpsPool) Type() string {
	return common.CONSENSUS_PEER_MSG
}

func (p *bpsPool) Run(pid peer.ID, message *pb.Message) error {
	peerMsg := msg.PeerMsg{}
	err := proto.Unmarshal(message.Data, &peerMsg)
	if err != nil {
		p.log.Errorf("failed to unmarshal message from p2p, err:%s", err)
		return err
	}

	pubkey, err := crypto.UnmarshalPublicKey(peerMsg.PubKey)
	if err != nil {
		p.log.Errorf("failed to unserialize pubkey from %s, err:%s", err, peerMsg.Type)
	}

	p.add(pid, pubkey)

	// If is new bp, response self peer id and public key
	if peerMsg.Type == NEW_BP_REG {
		pubkeyBytes, err := p.self.pubKey.Bytes()
		if err != nil {
			p.log.Errorf("failed serialize pubkey as bytes, err:%s", err)
			return err
		}

		data, err := proto.Marshal(&msg.PeerMsg{
			Type:   OLD_BP_RESP,
			Id:     []byte(peer.ID(p.self.id)),
			PubKey: pubkeyBytes,
		})
		if err != nil {
			p.log.Errorf("failed to marshal peer message, err:%s", err)
			return err
		}
		go p.event.Post(&p2p.SendMsgEvent{
			Target: peer.ID(peerMsg.Id),
			Typ:    common.CONSENSUS_PEER_MSG,
			Data:   data,
		})
	}
	return nil
}

func (p *bpsPool) Error(err error) {
	p.log.Errorf("bpsPool error: %s", err)
}
