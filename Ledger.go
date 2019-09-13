package main

import (
	"fmt"
	"strconv"
	"sync"
)

type Ledger struct {
	Accounts map[string]int
	lock     sync.Mutex
}

func MakeLedger() *Ledger {
	ledger := new(Ledger)
	ledger.Accounts = make(map[string]int)
	return ledger
}
func (Ledger Ledger) PrintLedger() {
	ledger.lock.Lock()
	defer ledger.lock.Unlock()
	for name, amount := range ledger.Accounts {
		fmt.Println(name + ": " + strconv.Itoa(amount))
	}
}
