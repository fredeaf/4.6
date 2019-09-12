package __6

import (
	"sync"
)

//Circle : structure containing addresses of network members
type Circle struct {
	M    map[int]string
	Lock sync.Mutex
}

//MakeDNS : dns initiator
func MakeCircle() *Circle {
	circle := new(Circle)
	circle.M = make(map[int]string)
	return circle
}

//AddConnection : adds a connection to dns
func (circle *Circle) AddPeer(address string) {
	circle.Lock.Lock()
	defer circle.Lock.Unlock()
	circle.M[len(circle.M)] = address
}

//RemoveConnection : Removes a connection from dns
func (circle *Circle) RemovePeer(address string) {
	circle.Lock.Lock()
	defer circle.Lock.Unlock()
	for k, x := range circle.M {
		if x == address {
			delete(circle.M, k)
		}
	}
}
