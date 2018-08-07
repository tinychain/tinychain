package tiny

import (
	"tinychain/p2p"
	"tinychain/event"
	"github.com/libp2p/go-libp2p-peer"
)

type Network interface {
	Start() error
	Stop() error
	AddProtocol(p2p.Protocol) error
	DelProtocol(p2p.Protocol)
}

// Network is the wrapper of physical p2p network layer
type Peer struct {
	config  *p2p.Config
	network *p2p.Peer
	event   *event.TypeMux

	// Send message event subscription
	sendSub      event.Subscription
	multiSendSub event.Subscription

	quitCh chan struct{}
}

func NewNetwork(config *p2p.Config) Network {
	network, err := p2p.New(config)
	if err != nil {
		log.Error("Failed to create p2p peer")
		return nil
	}
	return &Peer{
		config:  config,
		network: network,
		event:   event.GetEventhub(),
		quitCh:  make(chan struct{}),
	}
}

func (p *Peer) ID() peer.ID {
	return p.network.ID()
}

func (p *Peer) Start() error {
	p.sendSub = p.event.Subscribe(&p2p.SendMsgEvent{})
	p.multiSendSub = p.event.Subscribe(&p2p.MultiSendEvent{})
	go p.listen()
	return nil
}

func (p *Peer) listen() {
	for {
		select {
		case ev := <-p.sendSub.Chan():
			msg := ev.(*p2p.SendMsgEvent)
			go p.network.Send(msg.Target, msg.Typ, msg.Data)
		case ev := <-p.multiSendSub.Chan():
			msg := ev.(*p2p.MulticastEvent)
			go p.network.Multicast(msg.Targets, msg.Typ, msg.Data)
		case p.quitCh:
			p.sendSub.Unsubscribe()
			return
		}
	}
}

func (p *Peer) Stop() error {
	close(p.quitCh)
	return nil
}

func (p *Peer) AddProtocol(proto p2p.Protocol) error {
	return p.network.AddProtocol(proto)
}

func (p *Peer) DelProtocol(proto p2p.Protocol) {
	p.network.DelProtocol(proto)
}
