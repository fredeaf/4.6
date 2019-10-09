package main

import (
	"bufio"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
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
var myPrivateKey *rsa.PrivateKey
var myPublicKey *rsa.PublicKey
var keyStore = MakeStore()

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
		circle = pack.Circle                                                           //Circle is updated
		circle.Announce(myAddr, myID, string(x509.MarshalPKCS1PublicKey(myPublicKey))) //Announces presence to all other peers
		for _, p := range circle.nextTenPeers(myAddr) {
			conn, _ := net.Dial("tcp", p)
			if conn != nil {
				dns.AddConnection(conn)
				go handleConnection(conn)
			}
		}
	}
	if pack.Address != "" {
		newPupId, err := x509.ParsePKCS1PublicKey([]byte(pack.key))
		keyStore.AddKey(pack.uuid, newPupId)
		if pack.NewComer {
			conn, _ := net.Dial("tcp", pack.Address)
			if conn != nil {
				enc := gob.NewEncoder(conn)
				initialPack := new(Package)
				if err != nil {
					fmt.Println(err)
				}
				circle.AddPeer(pack.Address)
				initialPack.Circle = circle
				initialPack.keyStore = keyStore
				enc.Encode(initialPack) //send circle to new peer
			}
		}
		circle.AddPeer(pack.Address)
		key, err := x509.ParsePKCS1PublicKey([]byte(pack.key))
		if err != nil {
			fmt.Println(err)
		}
		keyStore.AddKey(pack.uuid, key)

	}
}

func listenForConnections(ln net.Listener) {
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
					fmt.Println("Please input uuid of receiver:")
					to, err := reader.ReadString('\n')
					fmt.Println("Please input amount:")
					amountString, err := reader.ReadString('\n')
					from := string(x509.MarshalPKCS1PublicKey(myPublicKey))
					to = strings.Replace(to, "\n", "", -1)
					amountString = strings.Replace(amountString, "\n", "", -1)
					amount, err := strconv.Atoi(amountString)
					if err != nil {
						fmt.Println("Error! try again")
					} else {
						fmt.Println(to)
						fmt.Println(circle.Peers)
						toKey := string(x509.MarshalPKCS1PublicKey(keyStore.GetKey(to)))
						signedVal := from + toKey + strconv.Itoa(amount)
						signature, err := rsa.SignPKCS1v15(rand.Reader, myPrivateKey, crypto.SHA256, []byte(signedVal))
						if err != nil {
							fmt.Println(err)
						}
						transaction := createTransaction(myID, string(x509.MarshalPKCS1PublicKey(myPublicKey)), toKey, amount, string(signature))
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
			case "3\n":
				{
					fmt.Println("test:")
					for _, p := range circle.Peers {
						fmt.Println(p)

					}
				}
			case "4\n":
				{
					fmt.Println("dns test")
					for _, x := range dns.m {
						fmt.Println(x.RemoteAddr().String())
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
	circle = MakeCircle()
	newID, err := uuid.NewUUID() //generates unique id
	myID = uuid.UUID.String(newID)
	myPrivateKey, err = rsa.GenerateKey(rand.Reader, 3000)
	if err != nil {
		fmt.Println(err)
	}
	myPublicKey = &myPrivateKey.PublicKey
	tClock = MakeClock()
	gob.Register(Package{})
	gob.Register(Circle{})
	gob.Register(SignedTransaction{})
	gob.Register(KeyStore{})
	var reader = bufio.NewReader(os.Stdin) //Create reader to get user input
	fmt.Println("Please input an IP address and port number of known network member")
	address, err := reader.ReadString('\n') //Reads input
	if err != nil {
		fmt.Println(err)
		return
	}
	address = strings.Replace(address, "\n", "", -1)          //Trimming address
	conn, _ := net.DialTimeout("tcp", address, 5*time.Second) //Attempts connection to given address
	name, _ := os.Hostname()                                  //Find own name
	addrs, _ := net.LookupHost(name)                          //Find own address
	ln, _ := net.Listen("tcp", "")                            //Listen for incoming connections
	myAddr = ln.Addr().String()
	myAddr = addrs[0] + strings.Replace(myAddr, "[::]", "", -1) //add port to address
	fmt.Println("My address: " + myAddr)
	fmt.Println("my id: " + myID)
	circle.AddPeer(myAddr) //Adds self to Circle
	if conn != nil {
		//address responds
		dns.AddConnection(conn)
		go handleConnection(conn)
		go listenForConnections(ln)
		joinReq := new(Package)
		enc := gob.NewEncoder(conn)
		joinReq.Address = myAddr
		joinReq.NewComer = true
		joinReq.key = string(x509.MarshalPKCS1PublicKey(myPublicKey))
		joinReq.uuid = myID
		enc.Encode(joinReq)
		takeInputFromUser()
	} else {
		//address not responding
		go listenForConnections(ln)
		takeInputFromUser()
	}
}
