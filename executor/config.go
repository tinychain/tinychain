package executor

import "tinychain/common"

type Config struct {
	MaxGasLimit uint64
}

func newConfig(config *common.Config) *Config {
	return &Config{
		MaxGasLimit: uint64(config.GetInt64("max_gas_limit")),
	}
}
