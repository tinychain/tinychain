package dpos_bft

import "tinychain/common"

type Config struct {
	GasLimit uint64 // Block gas limit
	PrivKey  []byte // Private key of block producer
}

func newConfig(config *common.Config) *Config {
	return &Config{
		GasLimit: uint64(config.GetInt64("consensus.block_gas_limit")),
		PrivKey:  []byte(config.GetString("consensus.bp_private_key")),
	}
}
