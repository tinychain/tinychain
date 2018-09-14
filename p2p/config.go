package p2p

import (
	"github.com/libp2p/go-libp2p-crypto"
	ma "github.com/multiformats/go-multiaddr"
	"tinychain/common"
)

type Config struct {
	seeds         []ma.Multiaddr // Seed nodes for initialization
	routeFilePath string         // Store route table
	privKey       crypto.PrivKey // Private key of peer
	port          int            // Listener port
}

func newConfig(conf *common.Config) *Config {
	config := &Config{
		routeFilePath: conf.GetString(common.RouteFilePath),
		port:          conf.GetInt(common.Port),
	}

	privKeyStr := conf.GetString(common.NetPrivKey)
	privKey, err := crypto.UnmarshalPrivateKey([]byte(privKeyStr))
	if err != nil {
		panic("failed to decode private key")
	}
	config.privKey = privKey
	seeds := conf.GetSlice(common.Seeds)
	for _, seed := range seeds {
		ipfsAddr, err := ma.NewMultiaddr(seed)
		if err != nil {
			log.Errorf("invalid seed address %s", seed)
			continue
		}
		config.seeds = append(config.seeds, ipfsAddr)
	}
	return config
}
