package dpos_bft

import (
	"tinychain/p2p/pb"
	msg "tinychain/consensus/dpos_bft/message"
	"github.com/golang/protobuf/proto"
	"errors"
	"github.com/libp2p/go-libp2p-peer"
	"tinychain/common"
	"tinychain/core/types"
	"time"
	"tinychain/p2p"
	"fmt"
	"github.com/libp2p/go-libp2p-crypto"
	"bytes"
	"tinychain/core"
)

var (
	errPeerIdNotFound      = errors.New("invalid bp: it's peer ID is not found in selected BP set")
	errUnknownType         = errors.New("unknown message type")
	errDigestNotMatch      = errors.New("digest is invalid")
	errSignatureInvalid    = errors.New("signature is invalid")
	errReceiptNotMatch     = errors.New("receipt is not match the block header receiptHash")
	errProcessBlockTimeout = errors.New("process block timeout")
	errCommitTimeout       = errors.New("commit timeout")

	loopReadBlockGap     = 500 * time.Millisecond // read block gap in loop
	loopReadBlockTimeout = 10 * time.Second       // read block timeout
	commitTimeout        = 5 * time.Second        // commit timeout
)

// Type implements the `Protocol` interface, and returns the message type of consensus engine
func (eg *Engine) Type() string {
	return common.ConsensusMsg
}

// Run implements the `Protocol` interface, and handle the message received from p2p layer
func (eg *Engine) Run(pid peer.ID, message *pb.Message) error {
	consensusMsg := msg.ConsensusMsg{}
	err := proto.Unmarshal(message.Data, &consensusMsg)
	if err != nil {
		return err
	}

	var found bool
	// Check peer.ID is in BP set or not
	for _, bp := range eg.peerPool.getBPs() {
		if bp.id == pid {
			found = true
			break
		}
	}
	if !found {
		return errPeerIdNotFound
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

// Error implements the `Protocol` interface, and handle error from p2p layer
func (eg *Engine) Error(err error) {
	log.Errorf("consensus receive error from p2p layer, err:%s", err)
}

// fetchBlockLoop fetch block from block_pool in loop, and will return error if timeout
func (eg *Engine) fetchBlockLoop(seqNo uint64) (*types.Block, error) {
	ticker := time.NewTicker(loopReadBlockGap)
	timeout := time.NewTimer(loopReadBlockTimeout)
	for {
		select {
		case <-ticker.C:
			block := eg.blockPool.GetBlock(seqNo)
			if block != nil {
				return block, nil
			}
		case <-timeout.C:
			return nil, errors.New(fmt.Sprintf("wait for block #%d timeout", eg.SeqNo()))
		}
	}
}

// startBFT kicks off the bft process
// 1. retrived from block_pool, and multicast PRE_COMMIT
func (eg *Engine) startBFT() error {
	block, err := eg.fetchBlockLoop(eg.SeqNo())
	if err != nil {
		return err
	}
	hash := block.Hash()
	digest := common.Sha256(hash.Bytes()).Bytes()
	sign, err := eg.Self().PrivKey().Sign(digest)
	if err != nil {
		log.Errorf("failed to sign PRE_COMMIT message, err:%s", err)
		return err
	}
	pubKey, err := eg.Self().PubKey().Bytes()
	if err != nil {
		log.Errorf("failed to convert pubkey to bytes, err:%s", err)
		return err
	}
	eg.bftState.Store(PRE_COMMIT)
	return eg.multicastConsensus(&msg.ConsensusMsg{
		Type:      PRE_COMMIT,
		SeqNo:     eg.SeqNo(),
		Digest:    digest,
		PubKey:    pubKey,
		Signature: sign,
	})
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
	block, err := eg.fetchBlockLoop(message.SeqNo)
	if err != nil {
		log.Errorf("failed to fetch block from block_pool, err: %s", err)
		return err
	}

	// Check pre_commit info
	if err := eg.checkPreCommit(block, message); err != nil {
		log.Errorf("Check PRE_COMMIT not pass, err:%s", err)
		return err
	}

	go eg.event.Post(&core.ExecBlockEvent{block})

	timeout := time.NewTimer(eg.config.ProcessTimeout)
	select {
	case <-timeout.C:
		eg.nextBFTRound()
		return errProcessBlockTimeout
	case <-eg.execCompleteChan:
		// do nothing
	}

	// Check receipts have exist in consensus engine and match the block or not
	if receipts, ok := eg.receipts.Load(message.SeqNo); ok {
		if err := eg.validator.ValidateState(block, eg.state, receipts.(types.Receipts)); err != nil {
			log.Errorf("invalid block state, err:%s", err)
			return err
		}
	}

	digest, pubKey, sign, err := eg.computeConsensusInfo(block)
	if err != nil {
		return err
	}

	eg.bftState.Store(COMMIT)
	return eg.multicastConsensus(&msg.ConsensusMsg{
		Type:      COMMIT,
		SeqNo:     eg.SeqNo(),
		Digest:    digest,
		PubKey:    pubKey,
		Signature: sign,
	})
}

// commit receives commit message and decide whether to commit the block
func (eg *Engine) commit(message *msg.ConsensusMsg) error {
	eg.commitVotes += 1
	if eg.commitVotes <= eg.config.RoundSize*2/3 {
		return nil
	}

	block, err := eg.fetchBlockLoop(eg.SeqNo())
	if err != nil {
		log.Errorf("failed to fetch block from block_pool, err: %s", err)
		return err
	}

	go eg.event.Post(&core.CommitBlockEvent{
		Block: block,
	})

	timeout := time.NewTimer(commitTimeout)
	select {
	case ev := <-eg.commitCompleteSub.Chan():
		// Commit complete
		if block := ev.(*core.CommitCompleteEvent).Block; block.Height() != eg.SeqNo() {
			return errors.New(fmt.Sprintf("commit height is #%d, but current seqNo in bft process is #%d", block.Height(), eg.SeqNo()))
		}
		digest, pubkey, sign, err := eg.computeConsensusInfo(block)
		if err != nil {
			return err
		}
		if err := eg.multicastConsensus(&msg.ConsensusMsg{
			Type:      COMMIT,
			SeqNo:     eg.SeqNo(),
			Digest:    digest,
			PubKey:    pubkey,
			Signature: sign,
		}); err != nil {
			return err
		}
		eg.nextBFTRound()

		return eg.startBFT()
	case <-timeout.C:
		return errCommitTimeout
	}

}

func (eg *Engine) multicastConsensus(message *msg.ConsensusMsg) error {
	var pids []peer.ID
	for _, bp := range eg.peerPool.getBPs() {
		pids = append(pids, bp.id)
	}

	data, err := proto.Marshal(message)
	if err != nil {
		log.Errorf("failed to encode consensus msg, err:%s", err)
		return err
	}
	go eg.event.Post(&p2p.MulticastEvent{
		Targets: pids,
		Typ:     eg.Type(),
		Data:    data,
	})
	return nil
}

// checkPreCommits checks the PRE_COMMIT message is valid or not.
func (eg *Engine) checkPreCommit(block *types.Block, message *msg.ConsensusMsg) error {
	// Decode digest
	pubKey, err := crypto.UnmarshalPublicKey(message.PubKey)
	if err != nil {
		log.Errorf("invalid public key, err:%s", err)
		return err
	}

	// Compare digest with block.height
	localDigest := common.Sha256(block.Hash().Bytes())
	if bytes.Compare(localDigest.Bytes(), message.Digest) != 0 {
		return errDigestNotMatch
	}

	equal, err := pubKey.Verify(message.Digest, message.Signature)
	if err != nil {
		log.Errorf("error occurs when verify signature, err:%s", err)
		return err
	}
	if !equal {
		return errSignatureInvalid
	}

	return nil
}

func (eg *Engine) computeConsensusInfo(block *types.Block) (digest []byte, pubKey []byte, sign []byte, err error) {
	hash := block.Hash()
	digest = common.Sha256(hash.Bytes()).Bytes()
	sign, err = eg.Self().PrivKey().Sign(digest)
	if err != nil {
		log.Errorf("failed to sign PRE_COMMIT message, err:%s", err)
		return nil, nil, nil, err
	}
	pubKey, err = eg.Self().PubKey().Bytes()
	if err != nil {
		log.Errorf("failed to convert pubkey to bytes, err:%s", err)
		return nil, nil, nil, err
	}
	return digest, pubKey, sign, nil
}
