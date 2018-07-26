package solo

import "tinychain/common"

type Config struct {
	Address common.Address // block producer's address
	PrivKey []byte         // block producer's private key
	Extra   []byte         // Extra data that will be stored in a new proposed block
}

func newConfig(config *common.Config) *Config {

	conf := &Config{
		PrivKey: []byte(config.GetString("consensus.private_key")),
		Extra:   []byte(config.GetString("consensus.extra_data")),
	}
	addr := []byte(config.GetString("consensus.coinbase"))
	if addr != nil {
		conf.Address = common.DecodeAddr(addr)
	}
	return conf
}
