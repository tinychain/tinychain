package tiny

import (
	"tinychain/common"
	"tinychain/consensus"
	"tinychain/core/chain"
	"tinychain/core/executor"
	"tinychain/core/state"
	"tinychain/db"
	"tinychain/db/leveldb"
	"tinychain/event"
	"tinychain/consensus/solo"
	"tinychain/consensus/pow"
	"tinychain/consensus/vrf_bft"
	"fmt"
)

var (
	log = common.GetLogger("tinychain")
)

// Tiny implements the tinychain full node service
type Tiny struct {
	config *common.Config

	eventHub *event.TypeMux

	engine consensus.Engine

	chain *chain.Blockchain

	db *db.TinyDB

	state *state.StateDB

	network *Network

	executor executor.Executor
}

func New(config *common.Config) (*Tiny, error) {
	eventHub := event.GetEventhub()

	ldb, err := leveldb.NewLDBDataBase("tinychain")
	if err != nil {
		log.Errorf("Cannot create db, err:%s", err)
		return nil, err
	}
	// Create tiny db
	tinyDB := db.NewTinyDB(ldb)
	// Create state db
	statedb, err := state.New(ldb, nil)
	if err != nil {
		log.Errorf("cannot init state, err:%s", err)
		return nil, err
	}

	network := NewNetwork(config)
	bc, err := chain.NewBlockchain(ldb)
	if err != nil {
		log.Error("Failed to create blockchain")
		return nil, err
	}

	tiny := &Tiny{
		config:   config,
		eventHub: eventHub,
		db:       tinyDB,
		network:  network,
		chain:    bc,
		state:    statedb,
	}
	engineName := config.GetString(common.EngineName)
	blockValidator := executor.NewBlockValidator(config, bc)
	txValidator := executor.NewTxValidator(config, statedb)
	switch engineName {
	case common.SoloEngine:
		tiny.engine, err = solo.New(config, statedb, bc, blockValidator, txValidator)
	case common.PowEngine:
		tiny.engine, err = pow.New(config, statedb, bc, blockValidator, txValidator)
	case common.VrfBftEngine:
		tiny.engine, err = vrf_bft.New(config)
	default:
		return nil, fmt.Errorf("unknown consensus engine %s", engineName)
	}
	if err != nil {
		return nil, err
	}
	return tiny, nil
}

func (tiny *Tiny) Start() error {
	// Collect protocols and register in the protocol manager

	// start network
	tiny.network.Start()

}

func (tiny *Tiny) init() error {

}

func (tiny *Tiny) Close() {
	tiny.eventHub.Stop()
	tiny.network.Stop()
}
