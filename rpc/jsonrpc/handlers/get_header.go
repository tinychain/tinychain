package handlers

import (
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	"context"
	"tinychain/common"
	"tinychain/rpc/utils"
	"tinychain/tiny"
)

type getHeaderParams struct {
	Hash   string `json:"hash"`
	Height uint64 `json:"height"`
}

type getHeaderResult struct {
	Header utils.Header `json:"header"`
}

type GetHeaderHandler struct {
	tiny *tiny.Tiny
}

func (h GetHeaderHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (result interface{}, err *jsonrpc.Error) {
	var p getHeaderParams
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	hash := common.HexToHash(p.Hash)
	header := h.tiny.Chain.GetHeader(hash, p.Height)
	if header == nil {
		return nil, utils.ErrNotFound("header not found")
	}

	return getHeaderResult{
		Header: utils.Header{
			ParentHash:    header.ParentHash.Hex(),
			Height:        header.Height,
			StateRoot:     header.StateRoot.Hex(),
			TxRoot:        header.TxRoot.Hex(),
			ReceiptHash:   header.ReceiptsHash.Hex(),
			Coinbase:      header.Coinbase.Hex(),
			Time:          header.Time,
			GasUsed:       header.GasUsed,
			GasLimit:      header.GasLimit,
			Extra:         header.Extra,
			ConsensusInfo: header.ConsensusInfo,
		},
	}, nil
}

func (h GetHeaderHandler) Name() string {
	return "GetHeader"
}

func (h GetHeaderHandler) Params() interface{} {
	return getHeaderParams{}
}

func (h GetHeaderHandler) Result() interface{} {
	return getHeaderResult{}
}
