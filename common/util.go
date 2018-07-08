package common

import (
	"encoding/hex"
	"github.com/libp2p/go-libp2p-crypto"
	"encoding/binary"
)

func Hex(b []byte) []byte {
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], b)
	return enc
}

// Generate address by public key
func GenAddrByPubkey(key crypto.PubKey) (Address, error) {
	var addr Address
	pubkey, err := key.Bytes()
	if err != nil {
		return addr, err
	}
	pubkey = pubkey[1:]
	h := Sha256(pubkey)
	hash := h[len(h)-AddressLength:]
	addr = HashToAddr(Sha256(hash))
	return addr, nil
}

// Generate address by private key
func GenAddrByPrivkey(key crypto.PrivKey) (Address, error) {
	pubkey := key.GetPublic()
	return GenAddrByPubkey(pubkey)
}

func Uint2Bytes(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b[:]
}

func Bytes2Uint(d []byte) uint64 {
	return binary.BigEndian.Uint64(d)
}
