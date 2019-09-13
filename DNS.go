package main

import (
	"net"
	"sync"
)

//DNS : structure containing connections
type DNS struct {
	m    map[int]net.Conn
	lock sync.Mutex
}

//MakeDNS : dns initiator
func MakeDNS() *DNS {
	dns := new(DNS)
	dns.m = make(map[int]net.Conn)
	return dns
}

//AddConnection : adds a connection to dns
func (dns *DNS) AddConnection(connection net.Conn) {
	dns.lock.Lock()
	defer dns.lock.Unlock()
	if newConnection(dns.m, connection) {
		dns.m[len(dns.m)] = connection
	}
}

//simple reverse contains function for peers
func newConnection(connections map[int]net.Conn, new net.Conn) bool {
	for _, p := range connections {
		if p == new {
			return false
		}
	}
	return true
}

//RemoveConnection : Removes a connection from dns
func (dns *DNS) RemoveConnection(connection net.Conn) {
	dns.lock.Lock()
	defer dns.lock.Unlock()
	for k, x := range dns.m {
		if x == connection {
			delete(dns.m, k)
		}
	}
}
