package main

import (
	"sync"
)

//keystore : structure containing public keys of network members
type KeyStore struct {
	KeyMap map[string]string
	lock   sync.Mutex
}

//MakeStore : KeyStore initiator
func MakeStore() *KeyStore {
	store := new(KeyStore)
	store.KeyMap = make(map[string]string)
	return store
}

//AddKey : adds a Key to the store
func (store KeyStore) AddKey(uuid string, key string) {
	store.lock.Lock()
	defer store.lock.Unlock()
	store.KeyMap[uuid] = key
}

//GetKey : returns the Key of user with supplied Uuid
func (store KeyStore) GetKey(uuid string) string {
	store.lock.Lock()
	return store.KeyMap[uuid]
}

//GetUuid : returns the id of user with supplied key
func (store KeyStore) GetUuid(key string) (uuid string) {
	store.lock.Lock()
	defer store.lock.Unlock()
	for id, k := range store.KeyMap {
		if k == key {
			uuid = id
			return
		}
	}
	return
}
