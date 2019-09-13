package main

//Package : a container for Transactions with a counter
type Package struct {
	Number      int
	Transaction *Transaction
	Address     string
	Circle      *Circle
	NewComer    bool
}

//packTransaction : packages Transactions with a package myID
func packTransaction(transaction *Transaction) *Package {
	packagesSent++
	pack := new(Package)
	pack.Number = packagesSent
	pack.Transaction = transaction
	return pack
}
