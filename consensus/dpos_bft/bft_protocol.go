package dpos_bft

import (
	"tinychain/p2p/pb"
	msg "tinychain/consensus/dpos_bft/message"
	"github.com/golang/protobuf/proto"
	"errors"
	"github.com/libp2p/go-libp2p-peer"
	"tinychain/common"
)

var (
	errUnknownType = errors.New("unknown message type")
)

// Type implements the `Protocol` interface, and returns the message type of consensus engine
func (eg *Engine) Type() string {
	return common.CONSENSUS_MSG
}

// Run implements the `Protocol` interface, and handle the message received from p2p layer
func (eg *Engine) Run(pid peer.ID, message *pb.Message) error {
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
		eg.log.Errorf("error: %s", errUnknownType)
		return errUnknownType
	}
}

// Error implements the `Protocol` interface, and handle error from p2p layer
func (eg *Engine) Error(err error) {
	eg.log.Errorf("consensus receive error from p2p layer, err:%s", err)
}

// startBFT kicks off the bft process
// 1. propse next blocks
// 2. if valid, multicast PRE_COMMIT
func (eg *Engine) startBFT() {

}

// preCommit receives pre_commit message and decide whether to process the block
// and multicast COMMIT
// 1. process block
// 2. if valid, multicast COMMIT
func (eg *Engine) preCommit(message *msg.ConsensusMsg) error {
	eg.preCommitVotes += 1
	if eg.preCommitVotes <= eg.config.RoundSize*2/3 {
		return nil
	}



}

// commit receives commit message and decide whether to commit the block
func (eg *Engine) commit(message *msg.ConsensusMsg) error {
	eg.commitVotes += 1
	if eg.commitVotes <= eg.config.RoundSize*2/3 {
		return nil
	}


}
