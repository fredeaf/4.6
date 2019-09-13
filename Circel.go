package main

import (
	"encoding/gob"
	"net"
	"sort"
	"sync"
)

//Circle : structure containing addresses of network members
type Circle struct {
	Peers []string
	lock  sync.Mutex
}

//MakeCircle : Circle initiator
func MakeCircle() *Circle {
	circle := new(Circle)
	circle.Peers = make([]string, 0)
	return circle
}

//AddPeer : adds a peer to circle
func (circle *Circle) AddPeer(address string) {
	circle.lock.Lock()
	defer circle.lock.Unlock()
	if isNew(circle.Peers, address) {
		circle.Peers = append(circle.Peers, address)
		sort.Strings(circle.Peers)
	}
}

//simple reverse contains function for peers
func isNew(peers []string, new string) bool {
	for _, p := range peers {
		if p == new {
			return false
		}
	}
	return true
}

//RemovePeer : Removes a connection from Circle
func (circle *Circle) RemovePeer(address string) {
	circle.lock.Lock()
	defer circle.lock.Unlock()
	PeersLeft := circle.Peers[:0]
	for _, v := range circle.Peers {
		if v != address {
			PeersLeft = append(PeersLeft, v)
		}
	}
	circle.Peers = PeersLeft
}

//Announce : announces presence to whole circle
func (circle *Circle) Announce(addr string) {
	circle.lock.Lock()
	defer circle.lock.Unlock()
	pack := new(Package)
	pack.Address = addr
	for _, p := range circle.Peers {
		if p != addr {
			sendAddr(pack, p)
		}
	}
}

//sendAddr : helper function for Announce
func sendAddr(pack *Package, peer string) {
	conn, _ := net.Dial("tcp", peer)
	if conn != nil {
		defer conn.Close()
		enc := gob.NewEncoder(conn)
		enc.Encode(pack)
	}
}
