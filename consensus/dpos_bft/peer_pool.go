package dpos_bft

import (
	"sync"
	"tinychain/p2p/pb"
	msg "tinychain/consensus/dpos_bft/message"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/golang/protobuf/proto"
	"tinychain/event"
	"tinychain/p2p"
	"tinychain/common"
	"github.com/libp2p/go-libp2p-peer"
)

const (
	NEW_BP_REG  = iota
	OLD_BP_RESP
)

type Peer interface {
	ID() peer.ID
}

type peerPool struct {
	peers sync.Map // map[peer.ID]crypto.PubKey
	event *event.TypeMux
	bp    *blockProducer
}

func newPeerPool(bp *blockProducer) *peerPool {
	return &peerPool{
		bp:    bp,
		event: event.GetEventhub(),
	}
}

func (p *peerPool) get(id peer.ID) crypto.PubKey {
	if pubkey, exist := p.peers.Load(id); exist {
		return pubkey.(crypto.PubKey)
	}
	return nil
}

func (p *peerPool) add(id peer.ID, pubKey crypto.PubKey) {
	p.peers.Store(id, pubKey)
}

func (p *peerPool) del(id peer.ID) {
	p.peers.Delete(id)
}

func (p *peerPool) count() int {
	var count int
	p.peers.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

func (p *peerPool) Type() string {
	return common.CONSENSUS_PEER_MSG
}

func (p *peerPool) Run(pid peer.ID, message *pb.Message) error {
	peerMsg := msg.PeerMsg{}
	err := proto.Unmarshal(message.Data, &peerMsg)
	if err != nil {
		log.Errorf("failed to unmarshal message from p2p, err:%s", err)
		return err
	}

	pubkey, err := crypto.UnmarshalPublicKey(peerMsg.PubKey)
	if err != nil {
		log.Errorf("failed to unserialize pubkey from %s, err:%s", err, peerMsg.Type)
	}

	p.add(pid, pubkey)

	// If is new bp, response self peer id and public key
	if peerMsg.Type == NEW_BP_REG {
		pubkeyBytes, err := p.bp.pubKey.Bytes()
		if err != nil {
			log.Errorf("failed serialize pubkey as bytes, err:%s", err)
			return err
		}

		data, err := proto.Marshal(&msg.PeerMsg{
			Type:   OLD_BP_RESP,
			Id:     []byte(peer.ID(p.bp.id)),
			PubKey: pubkeyBytes,
		})
		if err != nil {
			log.Errorf("failed to marshal peer message, err:%s", err)
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

func (p *peerPool) Error(err error) {

}
