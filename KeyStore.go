package main

import (
	"crypto/rsa"
	"sync"
)

//keystore : structure containing public keys of network members
type KeyStore struct {
	keyMap map[string]*rsa.PublicKey
	lock   sync.Mutex
}

//MakeStore : KeyStore initiator
func MakeStore() *KeyStore {
	store := new(KeyStore)
	store.keyMap = make(map[string]*rsa.PublicKey)
	return store
}

//AddKey : adds a key to the store
func (store KeyStore) AddKey(uuid string, key *rsa.PublicKey) {
	store.lock.Lock()
	defer store.lock.Unlock()
	store.keyMap[uuid] = key
}

//GetKey : returns the key of user with supplied uuid
func (store KeyStore) GetKey(uuid string) *rsa.PublicKey {
	var pubKey *rsa.PublicKey
	store.lock.Lock()
	defer store.lock.Unlock()
	if key, found := store.keyMap[uuid]; found {
		pubKey = key
	}
	return pubKey
}
