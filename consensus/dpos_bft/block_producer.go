package dpos_bft

import (
	"tinychain/common"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-peer"
)

type blockProducer struct {
	id      peer.ID
	address common.Address
	privKey crypto.PrivKey
	pubKey  crypto.PubKey
}

func newBP(config *Config, id peer.ID, privKey crypto.PrivKey) (*blockProducer, error) {
	address, err := common.GenAddrByPrivkey(privKey)
	if err != nil {
		return nil, err
	}
	return &blockProducer{
		id:      id,
		address: address,
		privKey: privKey,
		pubKey:  privKey.GetPublic(),
	}, nil
}
