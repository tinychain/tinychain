package handlers

import "math/big"

type header struct {
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

type transaction struct {
	Nonce     uint64   `json:"nonce"`
	GasPrices uint64   `json:"gas_prices"`
	GasLimit  uint64   `json:"gas_limit"`
	Value     *big.Int `json:"value"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Payload   []byte   `json:"payload"`
}

type block struct {
	Header       header         `json:"header"`
	Transactions []*transaction `json:"transactions,omitempty"`
	Pubkey       string         `json:"pubkey"`
	Signature    string         `json:"signature"`
}
