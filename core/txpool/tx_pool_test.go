package txpool

import (
	"tinychain/core/state"
	"tinychain/db/leveldb"
	"testing"
	"tinychain/executor"
	"tinychain/event"
	"tinychain/core"
	"tinychain/core/types"
	"math/big"
	"tinychain/account"
	"github.com/magiconair/properties/assert"
)

var (
	txPool   *TxPool
	eventHub = event.GetEventhub()
)

func TestNewTxPool(t *testing.T) {
	db, _ := leveldb.NewLDBDataBase("")
	config := &Config{
		1000,
		20,
	}
	validateConfig := &executor.Config{
		MaxGasLimit: 1000,
	}
	state := state.New(db, nil)
	validator := executor.NewTxValidator(validateConfig, state)
	txPool = NewTxPool(config, validator, state)

	txPool.Start()
}

func GenTxEample(nonce uint64) *types.Transaction {
	acc, _ := account.NewAccount()
	return types.NewTransaction(
		nonce,
		10000,
		new(big.Int).SetUint64(0),
		nil,
		acc.Address,
		acc.Address,
	)
}

func TestTxPoolAdd(t *testing.T) {
	tx := GenTxEample(0)
	ev := &core.NewTxEvent{
		Tx: tx,
	}
	eventHub.Post(ev)

	pending := txPool.Pending()
	assert.Equal(t, pending[tx.From], tx)
}
