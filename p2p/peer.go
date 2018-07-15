package p2p

import (
	"context"
	"github.com/libp2p/go-libp2p-peer"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	libnet "github.com/libp2p/go-libp2p-net"
	ma "github.com/multiformats/go-multiaddr"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	"tinychain/p2p/pb"
	"time"
	"github.com/pkg/errors"
	"tinychain/common"
	"github.com/libp2p/go-libp2p-protocol"
	"fmt"
	"crypto/rand"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-swarm"
	"sync"
	"tinychain/event"
)

const (
	MaxRespBufSize  = 100
	MaxNearestPeers = 20
	DEFAULT_TIMEOUT = time.Second * 60
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
	mux        *event.TypeMux
	host       *bhost.BasicHost // Local peer host
	routeTable *RouteTable      // Local route table
	context    context.Context
	respCh     chan *pb.Message // Response channel. Receive message from stream.
	quitCh     chan struct{}    // quit channel
	protocols  sync.Map         // Handlers of upper layer. map[string][]*Protocol
	timeout    time.Duration    // Timeout of per connection
	ready      chan struct{}    // Physical network is ready or not
}

// Creates new peer struct
func New(config *Config) (*Peer, error) {
	host, err := newHost(config.port, config.privKey)
	if err != nil {
		log.Errorf("Cannot create host: %s", err)
		return nil, err
	}

	peer := &Peer{
		host:    host,
		context: context.Background(),
		respCh:  make(chan *pb.Message, MaxRespBufSize),
		quitCh:  make(chan struct{}),
		timeout: DEFAULT_TIMEOUT,
		mux:     event.GetEventhub(),
	}
	peer.routeTable = NewRouteTable(config, peer)

	return peer, nil
}

func (peer *Peer) ID() peer.ID {
	return peer.host.ID()
}

// Link to a unknown peer with its multiaddr
// and send handshake
func (peer *Peer) Link(addr ma.Multiaddr) {
	// TODO
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
func (peer *Peer) Send(pid peer.ID, typ string, data interface{}) error {
	if pid == peer.ID() {
		return ErrSendToSelf
	}
	err := peer.Connect(pid)
	if err != nil {
		return err
	}
	stream := NewStreamWithPid(pid, peer)
	//peer.Streams.AddStream(stream)
	err = stream.send(typ, data)
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

	go peer.ListenMsg()
}

func (peer *Peer) Stop() {
	if peer.host != nil {
		peer.host.Network().Close()
		peer.host.Close()
	}
	peer.routeTable.Stop()
	peer.quitCh <- struct{}{}
	log.Info("Peer stopped successfully.")
}

func (peer *Peer) ListenMsg() {
	for {
		select {
		case message := <-peer.respCh:
			//log.Infof("Receive message: Name:%s, data:%s \n", message.Name, message.Data)
			// Handler run
			if protocols, exist := peer.protocols.Load(message.Name); exist {
				for _, proto := range protocols.([]Protocol) {
					go proto.Run(message)
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

func (peer *Peer) Broadcast(pbName string, data interface{}) {
	for pid := range peer.routeTable.Peers() {
		if pid == peer.ID() {
			continue
		}
		err := peer.Send(pid, pbName, data)
		if err != nil {
			log.Errorf("failed to send %s msg to peer %s, %s", pbName, pid.Pretty(), err)
		}
	}
}

// Multicast retrieves nearest peers from route table and send msg to them.
func (peer *Peer) Multicast(pids []peer.ID, pbName string, data interface{}) {
	for _, pid := range pids {
		if pid == peer.ID() {
			continue
		}
		err := peer.Send(pid, pbName, data)
		if err != nil {
			log.Errorf("failed to send %s msg to peer %s, %s", pbName, pid.Pretty(), err)
		}
		// move the active pid up ahead of the route bucket
		go peer.routeTable.update(pid)
	}
}
