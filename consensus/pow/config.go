package pow

import (
	"tinychain/common"
)

type Config struct {
	Miner      bool
	PrivKey    []byte
	GasLimit   uint64
	Difficulty uint64
}

func newConfig(config *common.Config) *Config {

	return &Config{
		Miner:      config.GetBool(common.IsMiner),
		PrivKey:    []byte(config.GetString(common.ConsensusPrivKey)),
		GasLimit:   uint64(config.GetInt64(common.BlockGasLimit)),
		Difficulty: uint64(config.GetInt64(common.Difficulty)),
	}
}
