package solo

import "tinychain/common"

type Config struct {
	BP       bool   // this peer is block producer or not
	PrivKey  []byte // block producer's private key
	Extra    []byte // Extra data that will be stored in a new proposed block
	GasLimit uint64
}

func newConfig(config *common.Config) *Config {

	conf := &Config{
		PrivKey:  []byte(config.GetString(common.ConsensusPrivKey)),
		Extra:    []byte(config.GetString(common.ExtraData)),
		GasLimit: uint64(config.GetInt64(common.BlockGasLimit)),
	}
	conf.BP = config.GetBool(common.IsBP)
	return conf
}
