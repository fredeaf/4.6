package main

//Package : a container for Transactions with a counter
type Package struct {
	Number      int
	Transaction *SignedTransaction
	Address     string
	key         string
	Circle      *Circle
	NewComer    bool
}

//packTransaction : packages Transactions with a package myID
func packTransaction(transaction *SignedTransaction) *Package {
	packagesSent++
	pack := new(Package)
	pack.Number = packagesSent
	pack.Transaction = transaction
	return pack
}
