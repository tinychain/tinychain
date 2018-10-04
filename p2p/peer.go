package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p-crypto"
	libnet "github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-protocol"
	"github.com/libp2p/go-libp2p-swarm"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"sync"
	"time"
	"github.com/tinychain/tinychain/common"
	"github.com/tinychain/tinychain/event"
)

const (
	MaxRespBufSize  = 100
	MaxNearestPeers = 20
	MaxStreamNum    = 1000
	DefaultTimeout  = time.Second * 60
)

var (
	TransProtocol = protocol.ID("/chain/1.0.0.")
	log           = common.GetLogger("p2p")

	ErrSendToSelf = errors.New("Send message to self")
)

// NewHost construct a host of libp2p
func newHost(port int, privKey crypto.PrivKey) (*bhost.BasicHost, error) {
	var (
		priv crypto.PrivKey
		pub  crypto.PubKey
	)
	if privKey == nil {
		var err error
		//priv, pub, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
		priv, pub, err = crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		pubB64, _ := peer.IDFromPublicKey(pub)
		privB64, _ := B64EncodePrivKey(priv)

		log.Info("Private key not found in config file.")
		log.Info("Generate new key pair of privkey and pubkey by Ed25519:")
		log.Infof("Pubkey:%s\n", pubB64)
		log.Infof("Privkey:%s\n", privB64)
	} else {
		priv = privKey
		pub = privKey.GetPublic()
	}
	//privKey, _ := crypto.MarshalPrivateKey(priv)
	//data := crypto.ConfigEncodeKey(privKey)
	//log.Info(data)
	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	if err != nil {
		log.Infof("New multiaddr:%s\n", err)
		log.Error(err)
		return nil, err
	}

	pstore := pstore.NewPeerstore()
	pstore.AddPrivKey(pid, priv)
	pstore.AddPubKey(pid, pub)

	ctx := context.Background()
	n := swarm.NewSwarm(ctx, pid, pstore, nil)
	err = n.Listen(addr)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	opts := &bhost.HostOpts{
		NATManager: bhost.NewNATManager,
	}
	return bhost.NewHost(ctx, n, opts)
}

// Peer stands for a logical peer of tinychain's p2p layer
type Peer struct {
	context    context.Context
	mux        *event.TypeMux
	host       *bhost.BasicHost // Local peer host
	routeTable *RouteTable      // Local route table
	protocols  sync.Map         // Handlers of upper layer. map[string][]*Protocol
	timeout    time.Duration    // Timeout of per connection
	streamPool *lru.Cache       // pool of live stream

	respCh chan *innerMsg // Response channel. Receive message from stream.
	quitCh chan struct{}  // quit channel
	ready  chan struct{}  // Physical network is ready or not
}

// Creates new peer struct
func New(config *common.Config) (*Peer, error) {
	conf := newConfig(config)
	host, err := newHost(conf.port, conf.privKey)
	if err != nil {
		log.Errorf("failed to create host, %s", err)
		return nil, err
	}

	peer := &Peer{
		host:    host,
		context: context.Background(),
		respCh:  make(chan *innerMsg, MaxRespBufSize),
		quitCh:  make(chan struct{}),
		timeout: DefaultTimeout,
		mux:     event.GetEventhub(),
	}
	peer.routeTable = NewRouteTable(conf, peer)
	streamPool, err := lru.NewWithEvict(MaxStreamNum, func(key interface{}, value interface{}) {
		value.(*Stream).close(nil)
	})
	if err != nil {
		log.Errorf("failed to init stream pool, %s", err)
		return nil, err
	}
	peer.streamPool = streamPool

	return peer, nil
}

func (peer *Peer) ID() peer.ID {
	return peer.host.ID()
}

// Link to a unknown peer with its ipfs multiaddr
// and send handshake
func (peer *Peer) ConnectWithIPFS(addr ma.Multiaddr) error {
	// TODO
	pid, _, err := ParseFromIPFSAddr(addr)
	if err != nil {
		return err
	}
	return peer.Connect(pid)
}

// Connect to a peer
func (peer *Peer) Connect(pid peer.ID) error {
	ctx, cancel := context.WithTimeout(peer.context, peer.timeout) //TODO: configurable?
	defer cancel()
	err := peer.host.Connect(ctx, pstore.PeerInfo{ID: pid})
	if err != nil {
		log.Infof("Failed to connect to peer %s\n", pid.Pretty())
		return err
	}
	return nil
}

// Send message to a peer
func (peer *Peer) Send(pid peer.ID, typ string, data []byte) error {
	if pid == peer.ID() {
		return ErrSendToSelf
	}

	stream := NewStreamWithPid(pid, peer)
	err := stream.send(typ, data)
	if err != nil {
		return err
	}
	peer.routeTable.update(pid)
	return nil
}

func (peer *Peer) Start() {
	log.Infof("Peer start with pid %s. Listen on addr: %s.\n", peer.ID().Pretty(), peer.host.Addrs())
	// Listen to stream arriving
	peer.host.SetStreamHandler(TransProtocol, peer.onStreamConnected)

	// Sync route with seeds and neighbor
	peer.routeTable.Start()

	go peer.listenMsg()
}

func (peer *Peer) Stop() {
	if peer.host != nil {
		peer.host.Network().Close()
		peer.host.Close()
	}
	peer.routeTable.Stop()
	close(peer.quitCh)
	log.Info("Peer stopped successfully.")
}

func (peer *Peer) listenMsg() {
	for {
		select {
		case inner := <-peer.respCh:
			//log.Infof("Receive message: Name:%s, data:%s \n", message.Name, message.Data)
			// Handler run
			if protocols, exist := peer.protocols.Load(inner.msg.Name); exist {
				for _, proto := range protocols.([]Protocol) {
					go proto.Run(inner.pid, inner.msg)
				}
			}
		case <-peer.quitCh:
			break
		}
	}
}

func (peer *Peer) onStreamConnected(s libnet.Stream) {
	log.Infof("Receive stream. Peer is:%s\n", s.Conn().RemotePeer().String())
	stream := NewStream(s.Conn().RemotePeer(), s.Conn().RemoteMultiaddr(), s, peer)

	//peer.Streams.AddStream(stream)
	stream.start()
}

func (peer *Peer) Broadcast(pbName string, data []byte) {
	for pid := range peer.routeTable.Peers() {
		if pid == peer.ID() {
			continue
		}
		go func() {
			err := peer.Send(pid, pbName, data)
			if err != nil {
				log.Errorf("failed to send %s msg to peer %s, %s", pbName, pid.Pretty(), err)
			}
			peer.routeTable.update(pid)
		}()
	}
}

// Multicast retrieves nearest peers from route table and send msg to them.
func (peer *Peer) Multicast(pids []peer.ID, pbName string, data []byte) {
	for _, pid := range pids {
		if pid == peer.ID() {
			continue
		}
		go func() {
			err := peer.Send(pid, pbName, data)
			if err != nil {
				log.Errorf("failed to send %s msg to peer %s, %s", pbName, pid.Pretty(), err)
			}
			// move the active pid up ahead of the route bucket
			peer.routeTable.update(pid)
		}()
	}
}
