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
		PrivKey:  []byte(config.GetString("consensus.private_key")),
		Extra:    []byte(config.GetString("consensus.extra_data")),
		GasLimit: uint64(config.GetInt64("consensus.gas_limit")),
	}
	conf.BP = config.GetBool("consensus.isBP")
	return conf
}
