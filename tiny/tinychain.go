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
	engine := consensus.New()

	bc, err := chain.NewBlockchain(ldb)
	if err != nil {
		log.Error("Failed to create blockchain")
		return nil, err
	}

	return &Tiny{
		config:   config,
		eventHub: eventHub,
		db:       tinyDB,
		network:  network,
		chain:    bc,
		engine:   engine,
		state:    statedb,
	}, nil
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
