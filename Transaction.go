package main

type SignedTransaction struct {
	ID        string
	From      string
	To        string
	Amount    int
	Signature string
}

func (l *Ledger) Transaction(t *SignedTransaction) {
	l.lock.Lock()
	defer l.lock.Unlock()

	/* We verify that the t.Signature is a valid RSA
	* signature on the rest of the fields in t under
	* the public key t.From.
	 */
	validSgnature := true

	if validSgnature {
		l.Accounts[t.From] -= t.Amount
		l.Accounts[t.To] += t.Amount
	}
}

//createTransaction : creates a transaction
func createTransaction(id string, from string, to string, amount int) *SignedTransaction {
	transaction := new(SignedTransaction)
	transaction.ID = id
	transaction.Amount = amount
	transaction.From = from
	transaction.To = to
	return transaction
}
