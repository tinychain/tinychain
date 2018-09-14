package common

const (
	// P2P Message Type
	OkMsg         = "OkMsg"
	RouteSyncReq  = "RouteSyncReq"
	RouteSyncResp = "RouteSyncResp"

	ConsensusMsg     = "ConsensusMsg"
	ConsensusPeerMsg = "ConsensusPeerMsg"
	ProposeBlockMsg  = "ProposeBlockMsg"
	ReadyBlockMsg    = "ReadyBlockMsg"

	NewTxMsg = "NewTxMsg"

	// Consensus Type
	SoloEngine    = "solo"
	PowEngine     = "pow"
	VrfBftEngine  = "vrf_bft"
	DposBftEngine = "dpos_bft"

	/* ---- Configuration field ------ */
	// consensus
	EngineName       = "consensus.engine"
	IsMiner          = "consensus.miner"
	ConsensusPrivKey = "consensus.private_key"
	BlockGasLimit    = "consensus.block_gas_limit"
	Difficulty       = "consensus.difficulty"
	ExtraData        = "consensus.extra_data"
	IsBP             = "consensus.isBP"

	// P2P
	RouteFilePath = "p2p.route_file_path"
	Port          = "p2p.port"
	NetPrivKey    = "p2p.private_key"
	Seeds         = "p2p.seeds"

	// Transaction Pool
	MaxTxSize     = "txpool.max_tx_size"
	PriceBump     = "txpool.price_bump"
	BatchTimeout  = "txpool.batch_timeout"
	BatchCapacity = "txpool.batch_capacity"

	// Block Pool
	MaxBlockSize = "blockpool.max_block_size"
)
