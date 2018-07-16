package p2p

import (
	"tinychain/p2p/pb"
	"errors"
	"github.com/libp2p/go-libp2p-peer"
)

var (
	ErrDupHandler = errors.New("p2p handler duplicate")
)

// Protocol represents the callback handler
type Protocol interface {
	// Typ should match the message type
	Type() string

	// Run func handles the message from the stream
	Run(pid peer.ID, message *pb.Message) error

	// Error func handles the error returned from the stream
	Error(error)
}

func (p *Peer) AddProtocol(proto Protocol) error {
	if protocols, exist := p.protocols.Load(proto.Type()); exist {
		protocols := protocols.([]Protocol)
		for _, protocol := range protocols {
			if protocol == proto {
				return ErrDupHandler
			}
		}
		protocols = append(protocols, proto)
		p.protocols.Store(proto.Type(), protocols)
	} else {
		p.protocols.Store(proto.Type(), []Protocol{proto})
	}
	return nil
}

func (p *Peer) DelProtocol(proto Protocol) {
	if protocols, exist := p.protocols.Load(proto.Type()); exist {
		protocols := protocols.([]Protocol)
		for i, protocol := range protocols {
			if protocol == proto {
				protocols = append(protocols[:i], protocols[i+1:]...)
				break
			}
		}
		p.protocols.Store(proto.Type(), protocols)
	}
}
