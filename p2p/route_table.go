package p2p

import (
	"bufio"
	"fmt"
	"github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"os"
	"strings"
	"sync"
	"time"
	"tinychain/common"
	"tinychain/p2p/pb"
)

var (
	autoRefreshInterval = 1 * time.Hour
)

const (
	bucketSize      = 16
	loopTimePerSync = 8
)

type RouteTable struct {
	peer          *Peer // Local peer
	peerStore     peerstore.Peerstore
	routeTable    *kbucket.RoutingTable // k-bucket route table
	routeFilePath string                // Route table cache file
	seeds         []ma.Multiaddr        // Seed peers for bootstrap

	maxPeersCountForSync int
	quitCh               chan struct{}
}

func NewRouteTable(config *Config, peer *Peer) *RouteTable {
	localId := peer.ID()

	pstore := peerstore.NewPeerstore()
	table := &RouteTable{
		peer:      peer,
		peerStore: peer.host.Peerstore(),
		routeTable: kbucket.NewRoutingTable(
			bucketSize,
			kbucket.ConvertPeerID(localId),
			time.Second*30,
			pstore,
		),
		seeds:                config.seeds,
		maxPeersCountForSync: bucketSize,
		routeFilePath:        config.routeFilePath,
	}
	//table.routeTable.Update(localId)

	return table
}

// Start route table looping for route sync
func (table *RouteTable) Start() {
	log.Info("Start route table sync...")
	table.LoadRTableFromFile()
	go table.syncLoop()
}

// Stop route table if is looping sync
func (table *RouteTable) Stop() {
	log.Info("Stop route table.")
	close(table.quitCh)
}

func (table *RouteTable) Peers() map[peer.ID][]ma.Multiaddr {
	peers := make(map[peer.ID][]ma.Multiaddr)
	for _, pid := range table.routeTable.ListPeers() {
		peers[pid] = table.peerStore.Addrs(pid)
	}
	return peers
}

// Add peer to route table
func (table *RouteTable) AddPeerInfo(prettyID string, addrStr []string) error {
	if prettyID == table.peer.ID().Pretty() {
		return nil
	}

	pid, err := peer.IDB58Decode(prettyID)
	if err != nil {
		return nil
	}

	addrs := make([]ma.Multiaddr, len(addrStr))
	for i, v := range addrStr {
		addrs[i], err = ma.NewMultiaddr(v)
		if err != nil {
			return err
		}
	}

	log.Infof("A peer is founded with pid %s and addrs %s.\n", pid.Pretty(), addrs)
	table.AddPeerWithAddrs(pid, addrs)
	//table.onRouteTableChange()
	return nil
}

// Add peer to route table
func (table *RouteTable) AddPeer(pid peer.ID, addr ma.Multiaddr) {
	if pid == table.peer.ID() {
		return
	}
	log.Infof("Adding Peer:%s,%s\n", pid.Pretty(), addr.String())
	table.peerStore.AddAddr(pid, addr, peerstore.PermanentAddrTTL)
	table.update(pid)
}

// Add peer with []ma.Multiaddrs
func (table *RouteTable) AddPeerWithAddrs(pid peer.ID, addrs []ma.Multiaddr) {
	if pid == table.peer.ID() {
		return
	}
	if table.routeTable.Find(pid) != "" {
		table.peerStore.SetAddrs(pid, addrs, peerstore.PermanentAddrTTL)
	} else {
		table.peerStore.AddAddrs(pid, addrs, peerstore.PermanentAddrTTL)
	}
	table.update(pid)
}

func (table *RouteTable) AddIPFSPeer(addr ma.Multiaddr) error {
	id, addr, err := ParseFromIPFSAddr(addr)
	if err != nil {
		log.Errorf("Failed to parse ipfs addr:%s", err)
		return err
	}
	table.AddPeer(id, addr)
	return nil
}

// Add peers when get 'RouteSyncResp'
func (table *RouteTable) AddPeers(peers []*pb.PeerInfo) error {
	//if len(peers) > table.maxPeersCountForSync {
	//	//TODO select first maxPeersCount
	//}
	for _, v := range peers {
		err := table.AddPeerInfo(v.Id, v.Addrs)
		if err != nil {
			log.Errorf("Failed to add peerInfo %s\n", v.Id)
			return err
		}
	}
	return nil
}

func (table *RouteTable) GetNearestPeers(pid peer.ID) []peerstore.PeerInfo {
	peers := table.routeTable.NearestPeers(kbucket.ConvertPeerID(pid), table.maxPeersCountForSync)

	peerInfos := make([]peerstore.PeerInfo, len(peers))
	for i, v := range peers {
		peerInfos[i] = table.peerStore.PeerInfo(v)
	}
	return peerInfos
}

// Sync route table
func (table *RouteTable) SyncRouteWithSeeds() {
	// sync with seed peers
	for _, ipfsAddr := range table.seeds {
		pid, addr, err := ParseFromIPFSAddr(ipfsAddr)
		if err != nil {
			continue
		}

		table.AddPeer(pid, addr)
		table.SyncFromPeer(pid)
	}
}

// Looping sync route table with neighbor
func (table *RouteTable) SyncRouteFromNeighbor() {
	syncedPeers := make(map[peer.ID]bool)
	var wg sync.WaitGroup

	loopTime := loopTimePerSync
	for loopTime > 0 {
		loopTime -= 1

		// Generate random peer id
		pid, err := RandomGeneratePid()
		if err != nil {
			log.Infof("Cannot generate pid randomly:%s\n", err)
			break
		}
		peers := table.GetNearestPeers(pid)
		select {
		case <-table.quitCh:
			break
		default:
			for _, peerInfo := range peers {
				// The peer have been synced
				if syncedPeers[peerInfo.ID] {
					continue
				}
				go func() {
					wg.Add(1)
					defer wg.Done()
					err := table.SyncFromPeer(peerInfo.ID)
					if err != nil {
						log.Infof("Failed to sync with peer %s:%s\n", peerInfo.ID, err)
						return
					}
					syncedPeers[peerInfo.ID] = true
				}()
			}
			wg.Wait()
		}
		table.SaveRTableToFile()

		// Delay for response of route syncing
		time.Sleep(7200 * time.Millisecond)
	}
}

// Sync route table with a peer
// It send `RouteSyncReq` message and transfer nil data
func (table *RouteTable) SyncFromPeer(pid peer.ID) error {
	if pid == table.peer.ID() {
		return nil
	}
	//stream := table.peer.Streams.Find(pid)
	//if stream == nil {
	stream := NewStreamWithPid(pid, table.peer)

	//}

	return stream.send(common.RouteSyncReq, nil)
}

// Start sync route table looping
func (table *RouteTable) syncLoop() {
	// Sync with seeds
	table.SyncRouteWithSeeds()
	go table.SyncRouteFromNeighbor()
	go table.refreshLoop()
}

// Refresh route table with given interval
func (table *RouteTable) refreshLoop() {
	ticker := time.NewTicker(autoRefreshInterval)

	// Looping sync with neighbor
	for {
		select {
		case <-ticker.C:
			log.Info("Start a new loop of route sync...")
			go table.SyncRouteFromNeighbor()
		case <-table.quitCh:
			return
		}
	}
}

func (table *RouteTable) nearestPeers(id peer.ID, count int) []peer.ID {
	return table.routeTable.NearestPeers(kbucket.ConvertPeerID(id), count)
}

func (table *RouteTable) update(id peer.ID) {
	table.routeTable.Update(id)
}

// Load route table from file
func (table *RouteTable) LoadRTableFromFile() {
	file, err := os.Open(table.routeFilePath)
	if err != nil {
		log.Info("Local route table doesn't exist.")
		return
	}
	defer file.Close()

	sc := bufio.NewScanner(file)
	sc.Split(bufio.ScanLines)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}

		ipfsaddr, err := ma.NewMultiaddr(line)
		if err != nil {
			log.Infof("Invalid ipfs addr format: %s\n", ipfsaddr)
			continue
		}
		err = table.AddIPFSPeer(ipfsaddr)
		if err != nil {
			continue
		}
	}
}

// Save route table to file
func (table *RouteTable) SaveRTableToFile() error {
	file, err := os.Create(table.routeFilePath)
	if err != nil {
		log.Errorf("Failed to create route table file: %s\n", err)
		return err
	}
	defer file.Close()

	file.WriteString(fmt.Sprintf("# Update Time: %s\n", time.Now().String()))
	peers := table.Peers()

	for pid, addrs := range peers {
		for _, addr := range addrs {
			line := fmt.Sprintf("%s/ipfs/%s\n", addr, pid.Pretty())
			file.WriteString(line)
		}
	}
	//log.Info("Save route table to local file.")
	return nil
}
