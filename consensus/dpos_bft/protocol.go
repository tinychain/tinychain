package dpos_bft

import (
	"tinychain/p2p/pb"
	msg "tinychain/consensus/dpos_bft/message"
	"github.com/golang/protobuf/proto"
	"errors"
)

var (
	errUnknownType = errors.New("unknown message type")
)

// Type implements the `Protocol` interface, and returns the message type of consensus engine
func (eg *Engine) Type() string {
	return "CONSENSUS_MSG"
}

// Run implements the `Protocol` interface, and handle the message received from p2p layer
func (eg *Engine) Run(message *pb.Message) error {
	consensusMsg := msg.ConsensusMsg{}
	err := proto.Unmarshal(message.Data, &consensusMsg)
	if err != nil {
		return err
	}

	switch consensusMsg.Type {
	case PRE_COMMIT:
		return eg.preCommit(&consensusMsg)
	case COMMIT:
		return eg.commit(&consensusMsg)
	default:
		log.Errorf("error: %s", errUnknownType)
		return errUnknownType
	}
}

// Error implements the `Protocol` interface
func (eg *Engine) Error(err error) {
	log.Errorf("consensus receive error from p2p layer, err:%s", err)
}

// startBFT kicks off the bft process
func (eg *Engine) startBFT() {

}

func (eg *Engine) preCommit(message *msg.ConsensusMsg) error {
	if eg.chain.LastBlock().Height() != message.SeqNo+1 {

	}
}

func (dpos *Engine) commit(message *msg.ConsensusMsg) error {

}
