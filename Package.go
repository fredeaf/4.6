package main

//Package : a container for Transactions with a counter
type Package struct {
	Transaction *SignedTransaction
	UUID        string
	Address     string
	Key         string
	Circle      *Circle
	NewComer    bool
	KeyStore    *KeyStore
	Sequencer   Sequencer
	Block       Block
}

//packTransaction : packages Transactions with a package myID
func packTransaction(transaction *SignedTransaction) *Package {
	pack := new(Package)
	pack.Transaction = transaction
	return pack
}

func packSequencer(sequencer *Sequencer) *Package {
	pack := new(Package)
	pack.Sequencer = *sequencer
	return pack
}
