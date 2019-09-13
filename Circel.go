package __6

import (
	"sort"
	"sync"
)

//Circle : structure containing addresses of network members
type Circle struct {
	Peers []string
	Lock  sync.Mutex
}

//MakeDNS : dns initiator
func MakeCircle() *Circle {
	circle := new(Circle)
	circle.Peers = make([]string, 0)
	return circle
}

//AddConnection : adds a connection to dns
func (circle *Circle) AddPeer(address string) {
	circle.Lock.Lock()
	defer circle.Lock.Unlock()
	circle.Peers = append(circle.Peers, address)
	sort.Strings(circle.Peers)
}

//RemoveConnection : Removes a connection from dns
func (circle *Circle) RemovePeer(address string) {
	circle.Lock.Lock()
	defer circle.Lock.Unlock()
	PeersLeft := circle.Peers[:0]
	for _, v := range circle.Peers {
		if v != address {
			PeersLeft = append(PeersLeft, v)
		}
	}
	circle.Peers = PeersLeft
}
