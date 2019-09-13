package main

type Transaction struct {
	ID     string
	From   string
	To     string
	Amount int
}

func (l *Ledger) Transaction(t *Transaction) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.Accounts[t.From] -= t.Amount
	l.Accounts[t.To] += t.Amount
}

//createTransaction : creates a transaction
func createTransaction(id string, from string, to string, amount int) *Transaction {
	transaction := new(Transaction)
	transaction.ID = id
	transaction.Amount = amount
	transaction.From = from
	transaction.To = to
	return transaction
}
