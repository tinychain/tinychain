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
)
