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
	m    map[string]int
	lock sync.Mutex
}

//MakeClock : clock initiator
func MakeClock(dns *DNS) *clock {
	clock := new(clock)
	clock.m = make(map[string]int)
	tClock.lock.Lock()
	defer tClock.lock.Unlock()
	dns.lock.Lock()
	defer dns.lock.Unlock()
	for _, con := range dns.m {
		clock.m[con.RemoteAddr().String()] = 0
	}
	return clock
}

func setClock(id string, num int) {
	tClock.lock.Lock()
	defer tClock.lock.Unlock()
	tClock.m[id] = num //TODO add logic
}
func checkClock(id string) bool {
	tClock.lock.Lock()
	defer tClock.lock.Unlock()
	return true //TODO add logic
}

func createTransaction(from string, to string, amount int) Transaction {
	return Transaction{
		ID:     "", //TODO create ID format/creation
		From:   from,
		To:     to,
		Amount: amount,
	}
}

func broadcast(transaction Transaction) {
	ledger.lock.Lock()
	defer ledger.lock.Unlock()
	if checkClock(transaction.ID) { //checks if transaction is new
		dns.lock.Lock()
		defer dns.lock.Unlock()
		for _, x := range dns.m {
			encoder := gob.NewEncoder(x)
			encoder.Encode(transaction) //new transaction is propagated
		}
		fmt.Println(ledger) //Updated ledger is printed
	} else {
	}
}

func handleConnection(conn net.Conn) {
	defer dns.RemoveConnection(conn)
	defer conn.Close()
	decoder := gob.NewDecoder(conn)
	var incoming Transaction
	for {
		otherEnd := conn.RemoteAddr().String() //Find address of connection
		for {
			err := decoder.Decode(&incoming)
			if err != nil {
				fmt.Println("Ending session with " + otherEnd)
				return
			}
			broadcast(incoming) //Data is translated to transaction and sent on
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
			go broadcast(transaction)
		}
	}
}

func main() {
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
		defer conn.Close()
		dns.AddConnection(conn)
		go handleConnection(conn)
		go turnIntoAServer()
		takeInputFromUser()
		tClock = MakeClock(dns)
	} else {
		//address not responding
		go turnIntoAServer()
		takeInputFromUser()
		tClock = MakeClock(dns)
	}
}
