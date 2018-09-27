package utils

import (
	"math/big"
)

type Header struct {
	ParentHash    string   `json:"parent_hash"`
	Height        uint64   `json:"height"`
	StateRoot     string   `json:"state_root"`
	TxRoot        string   `json:"tx_root"`
	ReceiptHash   string   `json:"receipt_hash"`
	Coinbase      string   `json:"coinbase"`
	Time          *big.Int `json:"time"`
	GasUsed       uint64   `json:"gas_used"`
	GasLimit      uint64   `json:"gas_limit"`
	Extra         []byte   `json:"extra"`
	ConsensusInfo []byte   `json:"consensus_info"`
}

type Transaction struct {
	Nonce     uint64   `json:"nonce"`
	GasPrices uint64   `json:"gas_prices"`
	GasLimit  uint64   `json:"gas_limit"`
	Value     *big.Int `json:"value"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Payload   []byte   `json:"payload"`
	PubKey    string   `json:"pub_key"`
	Signature string   `json:"signature"`
}

type Block struct {
	Header       Header         `json:"header"`
	Transactions []*Transaction `json:"transactions,omitempty"`
	Pubkey       string         `json:"pubkey"`
	Signature    string         `json:"signature"`
}

type Receipt struct {
	PostState       string `json:"post_state"`
	Status          bool   `json:"status"`
	TxHash          string `json:"tx_hash"`
	ContractAddress string `json:"contract_address"`
	GasUsed         uint64 `json:"gas_used"`
	Logs            []*Log `json:"logs"`
}

type Log struct {
	Address     string   `json:"address"`
	Topics      []string `json:"topics"`
	Data        []byte   `json:"data"`
	BlockHeight uint64   `json:"block_height"`
	TxHash      string   `json:"tx_hash"`
	TxIndex     uint     `json:"tx_index"`
	BlockHash   string   `json:"block_hash"`
	Index       uint     `json:"index"`
	Removed     bool     `json:"removed"`
}
