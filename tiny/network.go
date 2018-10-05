package tiny

import (
	"github.com/libp2p/go-libp2p-peer"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/event"
	"github.com/tinychain/tinychain/p2p"
)

// Network is the wrapper of physical p2p network layer
type Network struct {
	peer  *p2p.Peer
	event *event.TypeMux

	// Send message event subscription
	sendSub      event.Subscription
	multiSendSub event.Subscription

	quitCh chan struct{}
}

func NewNetwork(config *common.Config) *Network {
	peer, err := p2p.New(config)
	if err != nil {
		log.Error("Failed to create p2p Network")
		return nil
	}
	return &Network{
		peer:   peer,
		event:  event.GetEventhub(),
		quitCh: make(chan struct{}),
	}
}

func (p *Network) ID() peer.ID {
	return p.peer.ID()
}

func (p *Network) Start() {
	p.sendSub = p.event.Subscribe(&p2p.SendMsgEvent{})
	p.multiSendSub = p.event.Subscribe(&p2p.MulticastEvent{})
	go p.listen()
}

func (p *Network) listen() {
	for {
		select {
		case ev := <-p.sendSub.Chan():
			msg := ev.(*p2p.SendMsgEvent)
			go p.peer.Send(msg.Target, msg.Typ, msg.Data)
		case ev := <-p.multiSendSub.Chan():
			msg := ev.(*p2p.MulticastEvent)
			go p.peer.Multicast(msg.Targets, msg.Typ, msg.Data)
		case <-p.quitCh:
			p.sendSub.Unsubscribe()
			return
		}
	}
}

func (p *Network) Stop() {
	close(p.quitCh)
	p.peer.Stop()
}

func (p *Network) Peer() *p2p.Peer {
	return p.peer
}

func (p *Network) AddProtocol(proto p2p.Protocol) error {
	return p.peer.AddProtocol(proto)
}

func (p *Network) DelProtocol(proto p2p.Protocol) {
	p.peer.DelProtocol(proto)
}
