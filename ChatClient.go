package __6

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var dns = MakeDNS()
var ledger = MakeLedger()
var tClock *clock
var packagesSent int

//Package : a container for Transactions with a counter
type Package struct {
	number      int
	transaction *Transaction
}

//packTransaction : packages Transactions with a package id
func packTransaction(transaction *Transaction) Package {
	packagesSent++
	return Package{
		number:      packagesSent,
		transaction: transaction,
	}
}

//DNS : structure containing connections
type DNS struct {
	m    map[int]net.Conn
	lock sync.Mutex
}

//MakeDNS : dns initiator
func MakeDNS() *DNS {
	dns := new(DNS)
	dns.m = make(map[int]net.Conn)
	return dns
}

//AddConnection : adds a connection to dns
func (dns *DNS) AddConnection(connection net.Conn) {
	dns.lock.Lock()
	defer dns.lock.Unlock()
	dns.m[len(dns.m)] = connection
}

//RemoveConnection : Removes a connection from dns
func (dns *DNS) RemoveConnection(connection net.Conn) {
	dns.lock.Lock()
	defer dns.lock.Unlock()
	for k, x := range dns.m {
		if x == connection {
			delete(dns.m, k)
		}
	}
}

//clock : structure counting received transactions
type clock struct {
	m    map[string][]int
	lock sync.Mutex
}

//MakeClock : clock initiator
func MakeClock() *clock {
	clock := new(clock)
	clock.m = make(map[string][]int)
	return clock
}

//setClock : increments clock
func setClock(id string, packageNumber int) {
	tClock.m[id] = append(tClock.m[id], packageNumber)
}

//checkClock : Checks if a packageNumber has been received from an id before
func checkClock(id string, packageNumber int) bool {
	if contains(tClock.m[id], packageNumber) { //Checks if packageNumber was seen
		return true
	} else {
		return false
	}
}

//logic for checkClock
func contains(c []int, n int) bool {
	for _, x := range c {
		if x == n {
			return true
		}
	}
	return false
}

//createTransaction : creates a transaction
func createTransaction(from string, to string, amount int) *Transaction {
	transaction := new(Transaction)
	transaction.ID = "todo" //TODO create ID
	transaction.Amount = amount
	transaction.From = from
	transaction.To = to
	return transaction
}

func broadcast(pack Package) {
	dns.lock.Lock()
	defer dns.lock.Unlock()
	for _, x := range dns.m {
		encoder := gob.NewEncoder(x)
		encoder.Encode(pack) //new transaction is propagated
	}
	fmt.Println(ledger) //Updated ledger is printed

}

func handleConnection(conn net.Conn) {
	defer dns.RemoveConnection(conn)
	defer conn.Close()
	decoder := gob.NewDecoder(conn)
	var incoming Package
	for {
		otherEnd := conn.RemoteAddr().String() //Find address of connection
		for {
			err := decoder.Decode(&incoming)
			if err != nil {
				fmt.Println("Ending session with " + otherEnd)
				return
			}
			tClock.lock.Lock()
			defer tClock.lock.Unlock()
			if checkClock(incoming.transaction.ID, incoming.number) { //checks if transaction is new
				setClock(incoming.transaction.ID, incoming.number)
				ledger.Transaction(incoming.transaction)
				broadcast(incoming) //Package is sent onward
			}
		}
	}
}

func turnIntoAServer() {
	name, _ := os.Hostname()         //Find own name
	addrs, _ := net.LookupHost(name) //Find own address
	for indx, addr := range addrs {
		//Prints address
		fmt.Println("Address number " + strconv.Itoa(indx) + ": " + addr)
	}
	ln, _ := net.Listen("tcp", "") //Listen for incoming connections
	defer ln.Close()
	for {
		_, port, _ := net.SplitHostPort(ln.Addr().String()) //Find port used for connection
		fmt.Println("Listening on port " + port)
		conn, _ := ln.Accept() //Accept incoming TCP-connections
		dns.AddConnection(conn)
		go handleConnection(conn)
	}
}

func takeInputFromUser() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("Please choose an action: ")
		fmt.Println("Enter 1 to show ledger")
		fmt.Println("Enter 2 to input a transaction")
		ress, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Input error, try again")
		} else {
			switch ress {
			case "1\n":
				{
					fmt.Println("Please input sender:")
					from, err := reader.ReadString('\n')
					fmt.Println("Please input receiver:")
					to, err := reader.ReadString('\n')
					fmt.Println("Please input amount:")
					amountString, err := reader.ReadString('\n')
					from = strings.Replace(from, "\n", "", -1)
					to = strings.Replace(to, "\n", "", -1)
					amountString = strings.Replace(amountString, "\n", "", -1)
					amount, err := strconv.Atoi(amountString)
					if err != nil {
						fmt.Println("Error! try again")
					} else {
						transaction := createTransaction(from, to, amount)
						newPackage := packTransaction(transaction)
						go broadcast(newPackage)
					}
				}
			case "2\n":
				{
					fmt.Println("Current Ledger:")
					ledger.lock.Lock()
					defer ledger.lock.Unlock()
					for name, amount := range ledger.Accounts {
						fmt.Println(name + ": " + strconv.Itoa(amount))
					}
				}

			default:
				fmt.Println("invalid input")
			}
		}

	}
}

func main() {
	packagesSent = 0
	tClock = MakeClock()

	var reader = bufio.NewReader(os.Stdin) //Create reader to get user input
	fmt.Println("Please input an IP address and port number of known network")
	address, err := reader.ReadString('\n') //Reads input
	if err != nil {
		return
	}
	address = strings.Replace(address, "\n", "", -1)          //Trimming address
	conn, _ := net.DialTimeout("tcp", address, 5*time.Second) //Attempts connection to given address
	if conn != nil {
		//address responds
		dns.AddConnection(conn)
		go handleConnection(conn)
		go turnIntoAServer()
		takeInputFromUser()
	} else {
		//address not responding
		go turnIntoAServer()
		takeInputFromUser()
	}
}
