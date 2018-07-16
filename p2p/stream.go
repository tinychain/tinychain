package p2p

import (
	"github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	libnet "github.com/libp2p/go-libp2p-net"
	"fmt"

	"tinychain/p2p/pb"
	"errors"
	"time"
	"github.com/golang/protobuf/proto"
	"tinychain/common"
)

var (
	routeSyncTimeout = 45 * time.Second
	normalTimeout    = 30 * time.Second
	okTimeout        = 30 * time.Second

	ErrInvalidType         = errors.New("invalid data type of message")
	ErrMsgTypeNotMatchData = errors.New("message type is not match with data")
)

// innerMsg transfer the response message to protocols through channel
type innerMsg struct {
	pid peer.ID    // remote peer id
	msg *pb.Message // message received from remote peer
}

type Stream struct {
	remoteId   peer.ID       // Remote peer id
	remoteAddr ma.Multiaddr  // Remote peer multiaddr
	stream     libnet.Stream // Stream between peer and remote peer
	peer       *Peer         // Local peer
	//handshake  bool          // Id handshake successful or not

	//handshakeSuccessCh chan struct{} // Channel when handshake successfully
	pbChan      chan *pb.Message // Channel for message transfering
	quitWriteCh chan struct{}    // Channel for quiting
}

func NewStreamWithPid(pid peer.ID, peer *Peer) *Stream {
	return NewStream(pid, nil, nil, peer)
}

func NewStream(pid peer.ID, addr ma.Multiaddr, stream libnet.Stream, peer *Peer) *Stream {
	return &Stream{
		remoteId:   pid,
		remoteAddr: addr,
		stream:     stream,
		peer:       peer,
		//handshake:   false,
		pbChan:      make(chan *pb.Message, 2*1024),
		quitWriteCh: make(chan struct{}, 1),
	}
}

// Connect to remote peer
func (s *Stream) connect() error {
	stream, err := s.peer.host.NewStream(
		s.peer.context,
		s.remoteId,
		TransProtocol,
	)
	if err != nil {
		//log.Infof("Failed to connect remote peer %s:%s\n", s.remoteId.Pretty(), err)
		return err
	}
	s.stream = stream
	s.remoteAddr = stream.Conn().RemoteMultiaddr()
	//log.Infof("Connect to Peer. Info: %s\n", s.remoteAddr)

	s.start()

	return nil
}

// Check is handshake successful or not
//func (s *Stream) Handshake() bool {
//	return s.handshake
//}

func (s *Stream) String() string {
	addrStr := ""
	if s.remoteAddr != nil {
		addrStr = s.remoteAddr.String()
	}
	return fmt.Sprintf("Peer Stream:%s,%s\n", s.remoteId.Pretty(), addrStr)
}

func (s *Stream) Close(reason error) {
	//log.Info("Closing stream.")

	// Clean up
	//s.peer.Streams.Remove(s.remoteId)

	// Quit write channel
	//s.quitWriteCh <- struct{}{}

	if s.stream != nil {
		s.stream.Close()
		s.stream = nil
	}
}

func (s *Stream) start() {
	//log.Infof("Stream to %s starts loop\n", s.remoteId)
	//go s.writeLoop()
	go s.readLoop()
}

func (s *Stream) send(typ string, data []byte) error {
	if s.stream == nil {
		if err := s.connect(); err != nil {
			return err
		}
	}
	message, err := pb.NewMessage(typ, data)
	if err != nil {
		return ErrMsgTypeNotMatchData
	}

	// Set deadline
	s.SetReadDeadline(typ)

	// Write data to stream
	seri, _ := message.Serialize()
	_, err = s.stream.Write(seri)
	if err != nil {
		log.Infof("Failed to send message to peer %s. Message name:%s",
			s.remoteAddr, message.Name)
		return err
	}
	return nil
}

func (s *Stream) SetReadDeadline(name string) {
	if s.stream == nil {
		return
	}
	switch name {
	case common.ROUTESYNC_REQ:
		fallthrough
	case common.ROUTESYNC_RESP:
		s.stream.SetReadDeadline(time.Now().Add(routeSyncTimeout))
	case common.OK_MSG:
		s.stream.SetReadDeadline(time.Now().Add(okTimeout))
	default:
		s.stream.SetReadDeadline(time.Now().Add(normalTimeout))
	}
}

// Write message to stream
//func (s *Stream) WriteMessage(message *pb.Message) error {
//	data, _ := message.Serialize()
//	_, err := s.stream.Write(data)
//	if err != nil {
//		log.Infof("Failed to send message to peer %s. Message name:%s",
//			s.remoteAddr, message.Name)
//		return err
//	}
//	return nil
//}

func (s *Stream) readLoop() {
	if s.stream == nil {
		if err := s.connect(); err != nil {
			s.Close(err)
			return
		}
	}

	var (
		message *pb.Message
		dataLen uint32
		buf     = make([]byte, 1024*4)
		msgBuf  = make([]byte, 1024)
	)

	for {
		n, err := s.stream.Read(buf)
		if err != nil {
			s.Close(err)
			//log.Infof("Stream Close. %s.\n", err)
			return
		}
		msgBuf = append(msgBuf, buf[:n]...)

		if dataLen == 0 {
			if uint32(len(msgBuf)) < pb.DATA_LENGTH_SIZE {
				continue
			}
			dataLen, err = pb.BytesToUint32(msgBuf[:pb.DATA_LENGTH_SIZE])
			if err != nil {
				log.Errorf("Failed to read data length:%s\n", err)
				break
			}
		}
		// Reading data is not enough
		if uint32(len(msgBuf))-pb.DATA_LENGTH_SIZE < dataLen {
			continue
		}

		message, err = pb.DeserializeMsg(msgBuf)
		if err != nil {
			log.Errorf("Failed to deserialize message:%s\n", err)
			break
		}
		err = s.handleMsg(message)
		if err != nil {
			log.Info(err)
		}
		return
	}
}

//func (s *Stream) writeLoop() {
//	//handshakeTimeout := time.NewTicker(30 * time.Second)
//	//select {
//	//case <-handshakeTimeout.C:
//	//	// handshake timeout
//	//	return
//	//case <-s.handshakeSuccessCh:
//	//}
//
//	for {
//		select {
//		case <-s.quitWriteCh:
//			log.Info("Quit stream write loop")
//			return
//		case pb := <-s.pbChan:
//			s.WriteMessage(pb)
//		}
//	}
//}

// Handle message coming from remote peer
func (s *Stream) handleMsg(message *pb.Message) error {
	// Discover and update remote peer in local route table
	s.peer.routeTable.AddPeer(s.remoteId, s.remoteAddr)

	// Handle message
	pbName := message.Name
	log.Infof("Peer %s receive pb `%s`\n", s.peer.ID(), pbName)
	switch pbName {
	case common.OK_MSG:
		// success response
		s.Close(nil)
	case common.ROUTESYNC_REQ:
		// A peer wants your route table
		return s.onSyncRoute()
	case common.ROUTESYNC_RESP:
		s.Close(nil)
		// Update local route table
		return s.syncRoute(message.Data)
	default:
		// Message from other modules
		//log.Infof("Message content: %s\n", message.Data)
		s.peer.respCh <- &innerMsg{
			pid: s.stream.Conn().RemotePeer(),
			msg: message,
		}
		s.Close(nil)
	}
	return nil
}

// Sync route request handler
func (s *Stream) onSyncRoute() error {
	// Get nearest peers from route table
	peers := s.peer.routeTable.GetNearestPeers(s.remoteId)

	peerInfos := make([]*pb.PeerInfo, len(peers))
	for i, v := range peers {
		pinfo := &pb.PeerInfo{
			Id:    v.ID.Pretty(),
			Addrs: make([]string, len(v.Addrs)),
		}
		for j, addr := range v.Addrs {
			pinfo.Addrs[j] = addr.String()
		}
		peerInfos[i] = pinfo
	}
	data, err := proto.Marshal(&pb.PeerData{
		Peers: peerInfos,
	})
	if err != nil {
		return err
	}
	return s.send(common.ROUTESYNC_RESP, data)
}

// Receive `ROUTESYNC_RESP` and Update local route table
func (s *Stream) syncRoute(data []byte) error {
	peerData := &pb.PeerData{}
	err := proto.Unmarshal(data, peerData)
	if err != nil {
		log.Errorf("Failed to unmarshal bytes to peer data")
		return err
	}
	return s.peer.routeTable.AddPeers(peerData.Peers)
}
