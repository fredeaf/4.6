package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"strconv"
)

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

	validSignature := true
	pupId, err := x509.ParsePKCS1PublicKey([]byte(t.From))
	if err != nil {
		fmt.Println(err)
	}
	signedVal := t.From + t.To + strconv.Itoa(t.Amount)
	err = rsa.VerifyPKCS1v15(pupId, crypto.SHA256, []byte(signedVal), []byte(t.Signature))

	if err != nil {
		validSignature = false
	}

	if validSignature {
		l.Accounts[t.From] -= t.Amount
		l.Accounts[t.To] += t.Amount
	}
}

//createTransaction : creates a transaction
func createTransaction(id string, from string, to string, amount int, signature string) *SignedTransaction {
	transaction := new(SignedTransaction)
	transaction.ID = id
	transaction.Amount = amount
	transaction.From = from
	transaction.To = to
	transaction.Signature = signature
	return transaction
}
