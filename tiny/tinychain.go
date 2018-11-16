package tiny

import (
	"fmt"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/consensus"
	"github.com/tinychain/tinychain/consensus/pow"
	"github.com/tinychain/tinychain/consensus/solo"
	"github.com/tinychain/tinychain/consensus/vrf_bft"
	"github.com/tinychain/tinychain/core/chain"
	"github.com/tinychain/tinychain/core/executor"
	"github.com/tinychain/tinychain/core/state"
	"github.com/tinychain/tinychain/db"
	"github.com/tinychain/tinychain/p2p"
	"github.com/tinychain/tinychain/rpc/jsonrpc"
)

var (
	log = common.GetLogger("tinychain")
)

// Tiny implements the tinychain full node service
type Tiny struct {
	config *common.Config
	db     db.Database

	engine   consensus.Engine
	executor *executor.Executor
	state    *state.StateDB
	network  *Network
	chain    *chain.Blockchain
	tinyDB   *db.TinyDB
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

	peer, err := p2p.New(config)
	if err != nil {
		log.Error("Failed to create p2p Network")
		return nil, err
	}
	network := NewNetwork(peer)
	bc, err := chain.NewBlockchain(ldb)
	if err != nil {
		log.Error("Failed to create blockchain")
		return nil, err
	}

	tiny := &Tiny{
		config:  config,
		db:      ldb,
		network: network,
		chain:   bc,
		state:   statedb,
		tinyDB:  db.NewTinyDB(ldb),
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

	tiny.executor = executor.New(config, ldb, bc, tiny.engine)

	return tiny, nil
}

func (tiny *Tiny) Start() error {
	// Collect protocols and register in the protocol manager

	// start network
	tiny.network.Start()
	tiny.executor.Start()
	tiny.engine.Start()

	// start json rpc server
	jsonrpc.Start(tiny)

	return nil
}

func (tiny *Tiny) Config() *common.Config {
	return tiny.config
}

func (tiny *Tiny) RawDB() db.Database {
	return tiny.db
}

func (tiny *Tiny) DB() *db.TinyDB {
	return tiny.tinyDB
}

func (tiny *Tiny) Chain() *chain.Blockchain {
	return tiny.chain
}

func (tiny *Tiny) Network() *Network {
	return tiny.network
}

func (tiny *Tiny) StateDB() *state.StateDB {
	return tiny.state
}

func (tiny *Tiny) Executor() *executor.Executor {
	return tiny.executor
}

func (tiny *Tiny) Close() {
	tiny.engine.Stop()
	tiny.executor.Stop()
	tiny.network.Stop()
}
