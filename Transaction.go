package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
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
	* the public Key t.From.
	 */

	validSignature := true
	pubKey, err := x509.ParsePKCS1PublicKey([]byte(t.From))
	if err != nil {
		fmt.Println(err)
	}
	signedVal := t.ID + t.From + t.To + strconv.Itoa(t.Amount)
	hashedSVal := sha256.Sum256([]byte(signedVal))
	err = rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashedSVal[:], []byte(t.Signature))

	if err != nil {
		fmt.Println("signature didnt verify")
		validSignature = false
	}

	if validSignature {
		l.Accounts[t.From] -= t.Amount
		l.Accounts[t.To] += t.Amount - 1
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
