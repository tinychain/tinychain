package common

import (
	"github.com/spf13/viper"
	"sync"
	"fmt"
)

const (
	MAX_GAS_LIMIT="max_gas_limit"
	MAX_BLOCK_SIZE="max_block_size"

)

type Config struct {
	conf *viper.Viper
	mu   sync.RWMutex
}

func NewConfig(path string) *Config {
	vp := viper.New()
	vp.SetConfigFile(path)
	err := vp.ReadInConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to read config from disk, err:%s", err))
	}
	return &Config{
		conf: vp,
	}
}

func (c *Config) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conf.Get(key)
}

func (c *Config) GetInt64(key string) int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conf.GetInt64(key)
}

func (c *Config) GetString(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conf.GetString(key)
}

func (c *Config) GetBool(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conf.GetBool(key)
}
