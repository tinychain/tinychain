package dpos_bft

import (
	"tinychain/common"
	"github.com/libp2p/go-libp2p-crypto"
)

type blockProducer struct {
	address common.Address
	privKey crypto.PrivKey
	pubKey  crypto.PubKey
}

func newBP(config *Config) (*blockProducer, error) {
	privKey, err := crypto.UnmarshalPrivateKey(config.PrivKey)
	if err != nil {
		return nil, err
	}
	address, err := common.GenAddrByPrivkey(privKey)
	if err != nil {
		return nil, err
	}
	return &blockProducer{
		address: address,
		privKey: privKey,
		pubKey:  privKey.GetPublic(),
	}, nil
}
