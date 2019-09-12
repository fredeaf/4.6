package __6

import (
	"net"
	"sync"
)

//Circle : structure containing addresses of network members
type Circle struct {
	m    map[int]net.Addr
	lock sync.Mutex
}

//MakeDNS : dns initiator
func MakeCircle() *Circle {
	circle := new(Circle)
	circle.m = make(map[int]net.Addr)
	return circle
}

//AddConnection : adds a connection to dns
func (circle *Circle) AddPeer(address net.Addr) {
	circle.lock.Lock()
	defer circle.lock.Unlock()
	circle.m[len(circle.m)] = address
}

//RemoveConnection : Removes a connection from dns
func (circle *Circle) RemovePeer(address net.Addr) {
	circle.lock.Lock()
	defer circle.lock.Unlock()
	for k, x := range circle.m {
		if x == address {
			delete(circle.m, k)
		}
	}
}
