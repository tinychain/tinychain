package account

import (
	"github.com/libp2p/go-libp2p-crypto"
	"crypto/rand"
	"github.com/tinychain/tinychain/common"
)

type Key struct {
	privKey crypto.PrivKey
	pubKey crypto.PubKey
}

func NewKeyPairs() (*Key, error) {
	priv, pub, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Key{priv, pub}, nil
}

func validatePrivKey(address common.Address, priv crypto.PrivKey) bool {
	addr, err := common.GenAddrByPrivkey(priv)
	if err != nil {
		return false
	}
	return addr == address
}
