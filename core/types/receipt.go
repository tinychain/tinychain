package types

import (
	"tinychain/common"
	json "github.com/json-iterator/go"
	"tinychain/core/bmt"
	"strconv"
)

// Receipt represents the results of a transaction
type Receipt struct {
	// Consensus fields
	PostState       common.Hash    `json:"root"`             // post state root
	Status          bool           `json:"status"`           // Transaction executing success or failed
	TxHash          common.Hash    `json:"tx_hash"`          // Transaction hash
	ContractAddress common.Address `json:"contract_address"` // Contract address
	GasUsed         uint64         `json:"gas_used"`         // gas used of transaction
}

func NewRecipet(root common.Hash, status bool, txHash common.Hash, gasUsed uint64) *Receipt {
	return &Receipt{
		PostState: root,
		Status:    status,
		TxHash:    txHash,
		GasUsed:   gasUsed,
	}
}

func (re *Receipt) SetContractAddress(addr common.Address) {
	re.ContractAddress = addr
}

func (re *Receipt) Serialize() ([]byte, error) {
	return json.Marshal(re)
}

func (re *Receipt) Deserialize(d []byte) error {
	return json.Unmarshal(d, re)
}

type Receipts []*Receipt

func (rps Receipts) Hash() common.Hash {
	receiptSet := bmt.WriteSet{}
	for i, receipt := range rps {
		data, _ := receipt.Serialize()
		hash := common.Sha256(data)
		receiptSet[strconv.Itoa(i)] = hash.Bytes()
	}
	root, _ := bmt.Hash(receiptSet)
	return root
}

func (rps Receipts) Serialize() ([]byte, error) {
	return json.Marshal(rps)
}

func (rps Receipts) Deserialize(d []byte) error {
	return json.Unmarshal(d, &rps)
}
