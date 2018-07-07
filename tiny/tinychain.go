package tiny

import (
	"tinychain/event"
	"tinychain/consensus"
	"tinychain/core"
	"tinychain/db"
	"tinychain/common"
	"tinychain/executor"
	"tinychain/db/leveldb"
	"tinychain/core/state"
)

var (
	log = common.GetLogger("tinychain")
)

// Tiny implements the tinychain full node service
type Tiny struct {
	config *Config

	eventHub *event.TypeMux

	engine consensus.Engine

	chain *core.Blockchain

	db *db.TinyDB

	state *state.StateDB

	network Network

	executor executor.Executor

	pm *ProtocolManager
}

func New(config *Config) (*Tiny, error) {
	eventHub := event.GetEventhub()

	ldb, err := leveldb.NewLDBDataBase("tinychain")
	if err != nil {
		log.Errorf("Cannot create db, %s", err)
		return nil, err
	}
	// Create tiny db
	tinyDB := db.NewTinyDB(ldb)
	// Create state db
	statedb := state.New(ldb, nil)

	network := NewNetwork(config.p2p)
	engine := consensus.New()

	bc, err := core.NewBlockchain(tinyDB, engine)
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
		pm:       NewProtocolManager(network),
	}, nil
}

func (chain *Tiny) Start() error{
	// Collect protocols and register in the protocol manager

	// start network
	err := chain.network.Start()
}

func (chain *Tiny) Stop() {
	chain.eventHub.Stop()
	chain.network.Stop()
}
