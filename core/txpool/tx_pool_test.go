package txpool

import (
	"github.com/tinychain/tinychain/core/state"
	"github.com/tinychain/tinychain/db"
	"testing"
	"github.com/tinychain/tinychain/core/executor"
	"github.com/tinychain/tinychain/event"
	"github.com/tinychain/tinychain/core"
	"github.com/tinychain/tinychain/core/types"
	"math/big"
	"github.com/tinychain/tinychain/account"
	"github.com/magiconair/properties/assert"
)

var (
	txPool   *TxPool
	eventHub = event.GetEventhub()
)

func TestNewTxPool(t *testing.T) {
	db, _ := db.NewLDBDataBase("")
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
