package p2p

import "github.com/libp2p/go-libp2p-peer"

/*
	Network events
 */

// DiscvActive will be throw when peers discover run 6 rounds
type DiscvActiveEvent struct{}

// Message sent with p2p network layer
type SendMsgEvent struct {
	Target peer.ID // Target peer id
	Typ    string  // Message type
	Data   []byte  // Message data
}

// Multicast msg with p2p network layer
type MulticastEvent struct {
	Targets []peer.ID
	Typ     string
	Data    []byte
}

// Multicast msg to `count` nearest neighbor
type MulticastNeighborEvent struct {
	Typ   string
	Data  []byte
	Count int
}

// Random Multicast message with p2p network layer
type RandomSendEvnet struct {
	Typ  string
	Data []byte
}

type BroadcastEvent struct {
	Typ  string
	Data []byte
}
