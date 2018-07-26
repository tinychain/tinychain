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
		MaxTxSize:     uint64(config.GetInt64("txpool.max_tx_size")),
		PriceBump:     config.GetInt("txpool.price_bump"),
		BatchTimeout:  config.GetDuration("txpool.batch_timeout"),
		BatchCapacity: config.GetInt("txpool.batch_capacity"),
	}
}
