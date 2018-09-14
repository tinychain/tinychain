package txpool

import (
	"time"
	"tinychain/common"
)

type Config struct {
	MaxTxSize     uint64 // Max size of tx pool
	PriceBump     int    // Price bump to decide whether to replace tx or not
	BatchTimeout  time.Duration
	BatchCapacity int
}

func newConfig(config *common.Config) *Config {
	return &Config{
		MaxTxSize:     uint64(config.GetInt64(common.MaxTxSize)),
		PriceBump:     config.GetInt(common.PriceBump),
		BatchTimeout:  config.GetDuration(common.BatchTimeout),
		BatchCapacity: config.GetInt(common.BatchCapacity),
	}
}
