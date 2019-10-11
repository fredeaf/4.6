package main

//Package : a container for Transactions with a counter
type Package struct {
	Transaction *SignedTransaction
	Uuid        string
	Address     string
	Key         string
	Circle      *Circle
	NewComer    bool
	KeyStore    *KeyStore
}

//packTransaction : packages Transactions with a package myID
func packTransaction(transaction *SignedTransaction) *Package {
	pack := new(Package)
	pack.Transaction = transaction
	return pack
}
