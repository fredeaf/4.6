package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"github.com/google/uuid"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var dns = MakeDNS()
var ledger = MakeLedger()
var tClock *Clock
var packagesSent int
var myID string
var circle *Circle
var myAddr string

func broadcast(pack *Package) {
	dns.lock.Lock()
	defer dns.lock.Unlock()
	for _, x := range dns.m {
		encoder := gob.NewEncoder(x)
		err := encoder.Encode(pack) //new transaction is propagated
		if err != nil {
			fmt.Println("Error broadcasting to: " + x.RemoteAddr().String())
			fmt.Println(err)
		}
	}
}

func handleConnection(conn net.Conn) {
	defer dns.RemoveConnection(conn)
	defer conn.Close()
	otherEnd := conn.RemoteAddr().String() //Find address of connection
	for {
		pack := &Package{}
		dec := gob.NewDecoder(conn)
		err := dec.Decode(pack)
		if err != nil {
			fmt.Println(err)
			fmt.Println("Ending session with " + otherEnd)
			return
		}
		interpret(pack)
	}
}

//Interpret : function for checking the contents of received packages
func interpret(pack *Package) {
	if pack.Transaction != nil {
		if pack.Transaction.ID != myID {
			tClock.lock.Lock()
			defer tClock.lock.Unlock()
			if checkClock(pack.Transaction.ID, pack.Number) { //checks if transaction is new
				setClock(pack.Transaction.ID, pack.Number)
				ledger.Transaction(pack.Transaction) // Ledger is updated with new transaction
				fmt.Println("Ledger updated: ")
				ledger.PrintLedger() //Updated ledger is printed
				fmt.Println("Please input to choose an action: 1 for new transaction, 2 to show ledger ")
				broadcast(pack) //Package is sent onward
			}
		}
	}
	if pack.Circle != nil {
		circle = pack.Circle    //Circle is updated
		circle.Announce(myAddr) //Announces presence to all other peers
	}
	if pack.Address != "" {
		circle.AddPeer(pack.Address)
	}
}

func listenForConnections() {
	name, _ := os.Hostname()         //Find own name
	addrs, _ := net.LookupHost(name) //Find own address
	for indx, addr := range addrs {
		fmt.Println("Address number " + strconv.Itoa(indx) + ": " + addr) //Prints address
	}
	ln, _ := net.Listen("tcp", "") //Listen for incoming connections
	myAddr = ln.Addr().String()
	myAddr = addrs[0] + strings.Replace(myAddr, "[::]", "", -1)
	fmt.Println("My address: " + myAddr)
	circle.AddPeer(myAddr) //Adds self to
	defer ln.Close()
	for {
		conn, _ := ln.Accept() //Accept incoming TCP-connections
		dns.AddConnection(conn)
		go handleConnection(conn)
	}
}

func takeInputFromUser() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("Please input to choose an action: 1 for new transaction, 2 to show ledger ")
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
						transaction := createTransaction(myID, from, to, amount)
						newPackage := packTransaction(transaction)
						ledger.Transaction(transaction)
						go broadcast(newPackage)
					}
				}
			case "2\n":
				{
					fmt.Println("Current Ledger:")
					ledger.PrintLedger()
				}

			default:
				fmt.Println("invalid input")
			}
		}

	}
}

func main() {
	packagesSent = 0
	circle = MakeCircle()
	newID, err := uuid.NewUUID() //generates unique id
	myID = uuid.UUID.String(newID)
	tClock = MakeClock()
	gob.Register(Package{})
	gob.Register(Circle{})
	gob.Register(Transaction{})
	var reader = bufio.NewReader(os.Stdin) //Create reader to get user input
	fmt.Println("Please input an IP address and port number of known network member")
	address, err := reader.ReadString('\n') //Reads input
	if err != nil {
		fmt.Println(err)
		return
	}
	address = strings.Replace(address, "\n", "", -1)          //Trimming address
	conn, _ := net.DialTimeout("tcp", address, 5*time.Second) //Attempts connection to given address
	if conn != nil {
		//address responds
		dns.AddConnection(conn)
		go handleConnection(conn)
		go listenForConnections()
		takeInputFromUser()
	} else {
		//address not responding
		go listenForConnections()
		takeInputFromUser()
	}
}
