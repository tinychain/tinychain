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

// Multisend msg with p2p network layer
type MultiSendEvent struct {
	Targets []peer.ID
	Typ     string
	Data    []byte
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
