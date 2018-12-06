package account

import (
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/crypto"
)

type Key struct {
	privKey crypto.PrivateKey
	pubKey  crypto.PublicKey
}

func NewKeyPairs() (*Key, error) {
	priv, pub, err := crypto.NewKeyPair()
	if err != nil {
		return nil, err
	}
	return &Key{priv, pub}, nil
}

func validatePrivKey(address common.Address, priv crypto.PrivateKey) bool {
	addr, err := common.GenAddrByPrivkey(priv)
	if err != nil {
		return false
	}
	return addr == address
}
