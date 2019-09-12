package __6

//Package : a container for Transactions with a counter
type Package struct {
	number      int
	transaction *Transaction
}

//packTransaction : packages Transactions with a package myID
func packTransaction(transaction *Transaction) Package {
	packagesSent++
	return Package{
		number:      packagesSent,
		transaction: transaction,
	}
}
