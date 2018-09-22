package tiny

import (
	"fmt"
	"tinychain/common"
	"tinychain/consensus"
	"tinychain/consensus/pow"
	"tinychain/consensus/solo"
	"tinychain/consensus/vrf_bft"
	"tinychain/core/chain"
	"tinychain/core/executor"
	"tinychain/core/state"
	"tinychain/db"
)

var (
	log = common.GetLogger("tinychain")
)

// Tiny implements the tinychain full node service
type Tiny struct {
	config *common.Config
	db     db.Database

	Engine   consensus.Engine
	Executor executor.Executor
	State    *state.StateDB
	Network  *Network
	Chain    *chain.Blockchain
	TinyDB   *db.TinyDB
}

func New(config *common.Config) (*Tiny, error) {
	ldb, err := db.NewLDBDataBase("tinychain")
	if err != nil {
		log.Errorf("Cannot create db, err:%s", err)
		return nil, err
	}
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
		config:  config,
		db:      ldb,
		Network: network,
		Chain:   bc,
		State:   statedb,
		TinyDB:  db.NewTinyDB(ldb),
	}
	engineName := config.GetString(common.EngineName)
	blockValidator := executor.NewBlockValidator(config, bc)
	txValidator := executor.NewTxValidator(config, statedb)
	switch engineName {
	case common.SoloEngine:
		tiny.Engine, err = solo.New(config, statedb, bc, blockValidator, txValidator)
	case common.PowEngine:
		tiny.Engine, err = pow.New(config, statedb, bc, blockValidator, txValidator)
	case common.VrfBftEngine:
		tiny.Engine, err = vrf_bft.New(config)
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
	tiny.Network.Start()
	tiny.Executor.Start()
	tiny.Engine.Start()

	return nil
}

func (tiny *Tiny) init() error {
	return nil
}

func (tiny *Tiny) Close() {
	tiny.Engine.Stop()
	tiny.Executor.Stop()
	tiny.Network.Stop()
}
