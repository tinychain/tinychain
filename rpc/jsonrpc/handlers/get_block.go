package handlers

import (
	"context"
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	"tinychain/common"
	"tinychain/core/chain"
)

type getBlockParams struct {
	Height uint64 `json:"height"`
	Hash   []byte `json:"hash"`
}

type getBlockResult struct {
	Block block `json:"block"`
}

type GetBlockHandler struct{}

func (h GetBlockHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (result interface{}, err *jsonrpc.Error) {
	var (
		p   getBlockParams
		txs []*transaction
	)
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	hash := common.HexToHash(string(p.Hash))
	height := p.Height
	blk := chain.GetBlock(hash, height)

	for _, tx := range blk.Transactions {
		txs = append(txs, &transaction{
			Nonce:     tx.Nonce,
			GasPrices: tx.GasPrice,
			GasLimit:  tx.GasLimit,
			Value:     tx.Value,
			From:      tx.From.Hex(),
			To:        tx.To.Hex(),
			Payload:   tx.Payload,
		})
	}

	return getBlockResult{
		Block: block{
			Header: header{
				ParentHash:    blk.ParentHash().Hex(),
				Height:        blk.Height(),
				StateRoot:     blk.StateRoot().Hex(),
				TxRoot:        blk.TxRoot().Hex(),
				ReceiptHash:   blk.ReceiptsHash().Hex(),
				Coinbase:      blk.Coinbase().Hex(),
				Time:          blk.Time(),
				GasUsed:       blk.GasUsed(),
				GasLimit:      blk.GasLimit(),
				Extra:         blk.Extra(),
				ConsensusInfo: blk.ConsensusInfo(),
			},
			Transactions: txs,
			Pubkey:       common.Bytes2Hex(blk.PubKey),
			Signature:    common.Bytes2Hex(blk.Signature),
		},
	}, nil
}

func (h GetBlockHandler) Name() string {
	return "GetBlock"
}

func (h GetBlockHandler) Params() interface{} {
	return getBlockParams{}
}

func (h GetBlockHandler) Result() interface{} {
	return getBlockResult{}
}
