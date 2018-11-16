package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/tinychain/tinychain/p2p/pb"
	"math/big"
)

const (
	HashLength    = 32
	AddressLength = 20
)

type Hash [HashLength]byte

// BigToHash sets byte representation of b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BigToHash(b *big.Int) Hash { return BytesToHash(b.Bytes()) }

// HexToHash sets byte representation of s to hash.
// If b is larger than len(h), b will be cropped from the left.
func HexToHash(s string) Hash { return BytesToHash(FromHex(s)) }

func (h Hash) String() string {
	return string(h[:])
}

func (h Hash) Bytes() []byte {
	return h[:]
}

func (h Hash) Hex() string {
	return string(Hex(h[:]))
}

func (h Hash) Big() *big.Int {
	return new(big.Int).SetBytes(h.Bytes())
}

// Decode hash string with "0x...." format to Hash type
func DecodeHash(data []byte) Hash {
	dec := make([]byte, HashLength)
	hex.Decode(dec, data[2:])
	return BytesToHash(dec)
}

func BytesToHash(d []byte) Hash {
	var h Hash
	if len(d) > HashLength {
		d = d[:HashLength]
	}
	copy(h[:], d)
	return h
}

func (h Hash) Nil() bool {
	return h == Hash{}
}

func Sha256(d []byte) Hash {
	return sha256.Sum256(d)
}

type Address [AddressLength]byte

func (addr Address) String() string {
	return string(addr[:])
}

func (addr Address) Bytes() []byte {
	return addr[:]
}

func (addr Address) Hex() string {
	enc := make([]byte, len(addr)*2)
	hex.Encode(enc, addr[:])
	return "0x" + string(enc)
}

func (addr Address) Big() *big.Int {
	return new(big.Int).SetBytes(addr.Bytes())
}

func (addr Address) Nil() bool {
	return addr == Address{}
}

func BytesToAddress(b []byte) Address {
	var addr Address
	if len(b) > AddressLength {
		b = b[:AddressLength]
	}
	copy(addr[:], b)
	return addr
}

func BigToAddress(b *big.Int) Address {
	return BytesToAddress(b.Bytes())
}

func CreateAddress(addr Address, nonce uint64) Address {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, nonce)
	return BytesToAddress(Sha256(append(addr.Bytes(), buf...)).Bytes())
}

func HashToAddr(hash Hash) Address {
	return BytesToAddress(hash[:AddressLength])
}

// Decode address in hex format to common.Address
func HexToAddress(d string) Address {
	h := []byte(d)
	dec := make([]byte, AddressLength)
	if bytes.Compare(h[:2], []byte("0x")) == 0 {
		h = h[2:]
	}
	hex.Decode(dec, h)
	return BytesToAddress(dec)
}

// Protocol represents the callback handler
type Protocol interface {
	// Typ should match the message type
	Type() string

	// Run func handles the message from the stream
	Run(pid peer.ID, message *pb.Message) error

	// Error func handles the error returned from the stream
	Error(error)
}
