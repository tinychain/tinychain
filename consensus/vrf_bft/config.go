package vrf_bft

import (
	"tinychain/common"
	"time"
)

type Config struct {
	RoundSize      int           // max number of block producers for one round
	GasLimit       uint64        // Block gas limit
	PrivKey        []byte        // Private key of block producer
	Extra          []byte        // Extra data that will be stored in a new proposed block
	ProcessTimeout time.Duration // process timeout
}

func newConfig(config *common.Config) *Config {
	return &Config{
		GasLimit:       uint64(config.GetInt64("consensus.block_gas_limit")),
		PrivKey:        []byte(config.GetString("consensus.bp_private_key")),
		ProcessTimeout: config.GetDuration("consensus.process_timeout"),
		Extra:          []byte(config.GetString("consensus.extra_data")),
	}
}
