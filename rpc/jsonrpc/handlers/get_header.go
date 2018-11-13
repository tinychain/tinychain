package handlers

import (
	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
	"context"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/rpc/utils"
	"github.com/tinychain/tinychain/internal/api"
)

type getHeaderParams struct {
	Hash   string `json:"hash"`
	Height uint64 `json:"height"`
}

type getHeaderResult struct {
	Header utils.Header `json:"header"`
}

type GetHeaderHandler struct {
	api *api.ChainAPI
}

func (h GetHeaderHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (result interface{}, err *jsonrpc.Error) {
	var p getHeaderParams
	if err := jsonrpc.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	hash := common.HexToHash(p.Hash)
	header := h.api.GetHeaderByHash(hash)
	if header == nil {
		return nil, utils.ErrNotFound("header not found")
	}

	return getHeaderResult{
		Header: *header,
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
