package handlers

import (
	"context"
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	"tinychain/common"
	"tinychain/rpc/utils"
	"tinychain/tiny"
)

type getBlockParams struct {
	Height uint64 `json:"height"`
	Hash   string `json:"hash"`
}

type getBlockResult struct {
	Block utils.Block `json:"block"`
}

type GetBlockHandler struct {
	tiny *tiny.Tiny
}

func (h GetBlockHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	var (
		p   getBlockParams
		txs []*utils.Transaction
	)
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	hash := common.HexToHash(p.Hash)
	height := p.Height
	blk := h.tiny.Chain().GetBlock(hash, height)
	if blk == nil {
		return nil, &jsonrpc.Error{
			Code:    404,
			Message: "block not found",
		}
	}

	for _, tx := range blk.Transactions {
		txs = append(txs, &utils.Transaction{
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
		Block: utils.Block{
			Header: utils.Header{
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
