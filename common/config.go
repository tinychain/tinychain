package common

import (
	"fmt"
	"github.com/spf13/viper"
	"sync"
	"time"
)

const (
	MAX_GAS_LIMIT    = "max_gas_limit"
	MAX_EXTRA_LENGTH = "max_extra_length"
	MAX_BLOCK_NUM    = "max_block_num"
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

func (c *Config) GetInt(key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conf.GetInt(key)
}

func (c *Config) GetDuration(key string) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conf.GetDuration(key)
}

func (c *Config) GetSlice(key string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conf.GetStringSlice(key)
}
