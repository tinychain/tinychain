package types

import (
	"tinychain/common"
	"math/big"
	"sync/atomic"
	json "github.com/json-iterator/go"
	"encoding/binary"
	"encoding/hex"
	"github.com/libp2p/go-libp2p-crypto"
	"bytes"
)

// BNonce is a 64-bit hash which proves that a sufficient amount of
// computation has been carried out on a block
type BNonce [8]byte

func EncodeNonce(i uint64) BNonce {
	var n BNonce
	binary.BigEndian.PutUint64(n[:], i)
	return n
}

func (n BNonce) Uint64() uint64 {
	return binary.BigEndian.Uint64(n[:])
}

func (n BNonce) Hex() []byte {
	return common.Hex(n[:])
}

func (n BNonce) DecodeHex(b []byte) error {
	dec := make([]byte, len(n))
	_, err := hex.Decode(dec, b[2:])
	if err != nil {
		return err
	}
	n.SetBytes(dec)
	return nil
}

func (n BNonce) SetBytes(b []byte) {
	if len(b) > len(n) {
		b = b[:len(n)]
	}
	copy(n[:], b)
}

type Header struct {
	ParentHash    common.Hash    `json:"parent_hash"`              // Hash of parent block
	Height        uint64         `json:"height"`                   // Block height
	StateRoot     common.Hash    `json:"state_root"`               // State root
	TxRoot        common.Hash    `json:"tx_root"`                  // Transaction tree root
	ReceiptsHash  common.Hash    `json:"receipt_hash"`             // Receipts hash
	Coinbase      common.Address `json:"miner"`                    // Miner address who receives reward of this block
	Time          *big.Int       `json:"time"`                     // Timestamp
	GasUsed       uint64         `json:"gas"`                      // Total gas used
	GasLimit      uint64         `json:"gas_limit"`                // Gas limit of this block
	Extra         []byte         `json:"extra,omitempty"`          // Extra data
	ConsensusInfo []byte         `json:"consensus_info,omitempty"` // Extra consensus information
}

func (hd *Header) Hash() common.Hash {
	var (
		data        []byte
		heightBytes []byte // bytes of height
		gusedBytes  []byte // bytes of GasUsed
		glimitBytes []byte // bytes of GasLimit
	)

	if hd.Height != 0 {
		binary.BigEndian.PutUint64(heightBytes, hd.Height)
	}
	if hd.GasUsed != 0 {
		binary.BigEndian.PutUint64(gusedBytes, hd.GasUsed)
	}
	if hd.GasLimit != 0 {
		binary.BigEndian.PutUint64(glimitBytes, hd.GasLimit)
	}

	data = bytes.Join([][]byte{
		hd.ParentHash.Bytes(),
		heightBytes,
		hd.StateRoot.Bytes(),
		hd.TxRoot.Bytes(),
		hd.ReceiptsHash.Bytes(),
		hd.Coinbase.Bytes(),
		hd.Time.Bytes(),
		gusedBytes,
		glimitBytes,
		hd.Extra,
		hd.ConsensusInfo,
	}, nil)

	hash := common.Sha256(data)
	return hash
}

func (hd *Header) Serialize() ([]byte, error) { return json.Marshal(hd) }

func (hd *Header) Desrialize(d []byte) error { return json.Unmarshal(d, hd) }

type Block struct {
	Header       *Header      `json:"header"`
	Transactions Transactions `json:"transactions"`
	PubKey       []byte       `json:"pub_key"`
	Signature    []byte       `json:"signature"`
	hash         atomic.Value // Header hash cache
	size         atomic.Value // Block size cache
}

func NewBlock(header *Header, txs Transactions) *Block {
	block := &Block{
		Header:       header,
		Transactions: txs,
	}
	return block
}

func (bl *Block) TxRoot() common.Hash       { return bl.Header.TxRoot }
func (bl *Block) ReceiptsHash() common.Hash { return bl.Header.ReceiptsHash }
func (bl *Block) ParentHash() common.Hash   { return bl.Header.ParentHash }
func (bl *Block) Height() uint64            { return bl.Header.Height }
func (bl *Block) StateRoot() common.Hash    { return bl.Header.StateRoot }
func (bl *Block) Coinbase() common.Address  { return bl.Header.Coinbase }
func (bl *Block) Extra() []byte             { return bl.Header.Extra }
func (bl *Block) Time() *big.Int            { return bl.Header.Time }
func (bl *Block) GasUsed() uint64           { return bl.Header.GasUsed }
func (bl *Block) GasLimit() uint64          { return bl.Header.GasLimit }
func (bl *Block) ConsensusInfo() []byte     { return bl.Header.ConsensusInfo }

// Calculate hash of block
// Combine header hash and transactions hash, and sha256 it
func (bl *Block) Hash() common.Hash {
	if hash := bl.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	if bl.Header.TxRoot.Nil() {
		bl.Header.TxRoot = bl.Transactions.Hash()
	}
	hash := bl.Header.Hash()
	bl.hash.Store(hash)
	return hash
}

func (bl *Block) Size() uint64 {
	if size := bl.size.Load(); size != nil {
		return size.(uint64)
	}
	tmp, _ := bl.Serialize()
	bl.size.Store(len(tmp))
	return uint64(len(tmp))
}

func (bl *Block) Sign(priv crypto.PrivKey) ([]byte, error) {
	hash := bl.Hash()
	s, err := priv.Sign(hash.Bytes())
	if err != nil {
		return nil, err
	}
	bl.PubKey, err = priv.GetPublic().Bytes()
	if err != nil {
		return nil, err
	}
	bl.Signature = s
	return s, nil
}

func (bl *Block) Clone() *Block {
	return &Block{
		Header: &Header{
			ParentHash:    bl.ParentHash(),
			Height:        bl.Height(),
			StateRoot:     bl.StateRoot(),
			TxRoot:        bl.TxRoot(),
			ReceiptsHash:  bl.ReceiptsHash(),
			Coinbase:      bl.Coinbase(),
			Time:          bl.Time(),
			GasUsed:       bl.GasUsed(),
			GasLimit:      bl.GasLimit(),
			Extra:         bl.Extra(),
			ConsensusInfo: bl.ConsensusInfo(),
		},
		Transactions: bl.Transactions,
		PubKey:       bl.PubKey,
		Signature:    bl.Signature,
	}
}

func (bl *Block) Serialize() ([]byte, error) { return json.Marshal(bl) }

func (bl *Block) Deserialize(d []byte) error { return json.Unmarshal(d, bl) }

type Blocks []*Block

func (blks Blocks) Len() int {
	return len(blks)
}

func (blks Blocks) Less(i, j int) bool {
	return blks[i].Height() < blks[j].Height()
}

func (blks Blocks) Swap(i, j int) {
	blks[i], blks[j] = blks[j], blks[i]
}
