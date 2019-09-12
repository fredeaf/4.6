package __6

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
	fmt.Println(ledger) //Updated ledger is printed
}

func handleConnection(conn net.Conn) {
	defer dns.RemoveConnection(conn)
	defer conn.Close()
	otherEnd := conn.RemoteAddr().String() //Find address of connection
	pack := &Package{}
	dec := gob.NewDecoder(conn)
	for {
		err := dec.Decode(pack)
		if err != nil {
			fmt.Println("Ending session with " + otherEnd)
			return
		}
		interpret(pack)
	}
}

//Interpret : function for checking the contents of received packages
func interpret(pack *Package) {
	if pack.Transaction != nil {
		tClock.lock.Lock()
		defer tClock.lock.Unlock()
		if checkClock(pack.Transaction.ID, pack.Number) { //checks if transaction is new
			setClock(pack.Transaction.ID, pack.Number)
			ledger.Transaction(pack.Transaction)
			broadcast(pack) //Package is sent onward
		}
	}
	if pack.Circle != nil {
		circle = pack.Circle //Circle is updated
	}
	if pack.Address != "" {
		circle.Lock.Lock()
		defer circle.Lock.Unlock()
		circle.M[len(circle.M)] = pack.Address //TODO:: reformat circle to sorted list/slice
	}

}

func listenForConnections() {
	name, _ := os.Hostname()         //Find own name
	addrs, _ := net.LookupHost(name) //Find own address
	for indx, addr := range addrs {
		fmt.Println("Address number " + strconv.Itoa(indx) + ": " + addr) //Prints address
	}
	ln, _ := net.Listen("tcp", "") //Listen for incoming connections
	defer ln.Close()
	for {
		_, port, _ := net.SplitHostPort(ln.Addr().String()) //Find port used for connection
		fmt.Println("Listening on port " + port)
		conn, _ := ln.Accept() //Accept incoming TCP-connections
		dns.AddConnection(conn)
		circle.AddPeer(conn.RemoteAddr().String())
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
						transaction := createTransaction(myID, from, to, amount)
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
	newID, err := uuid.NewUUID() //generates unique id
	myID = uuid.UUID.String(newID)
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
		go listenForConnections()
		takeInputFromUser()
	} else {
		//address not responding
		go listenForConnections()
		takeInputFromUser()
	}
}
