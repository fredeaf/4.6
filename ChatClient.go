package main

import (
	"bufio"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
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
	defer circle.RemovePeer(conn.RemoteAddr().String())
	defer conn.Close()
	for {
		pack := &Package{}
		dec := gob.NewDecoder(conn)
		err := dec.Decode(pack)
		if err != nil {
			return
		}
		interpret(pack)
	}
}

//Interpret : function for checking the contents of received packages
func interpret(pack *Package) {
	if pack.Transaction != nil {
		if pack.Uuid != myID {
			tClock.lock.Lock()
			defer tClock.lock.Unlock()
			if checkClock(pack.Transaction.ID, pack.Uuid) { //checks if transaction is new
				setClock(pack.Transaction.ID, pack.Uuid)
				ledger.Transaction(pack.Transaction) // Ledger is updated with new transaction
				fmt.Println("Ledger updated: ")
				ledger.PrintLedger() //Updated ledger is printed
				fmt.Println("Please input to choose an action: 1 for new transaction, 2 to show ledger, 3 print Uuid ")
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
	if pack.KeyStore != nil {
		keyStore = pack.KeyStore
	}
	if pack.Address != "" {
		keyStore.AddKey(pack.Uuid, pack.Key)
		circle.AddPeer(pack.Address)
		if pack.NewComer {
			conn, _ := net.Dial("tcp", pack.Address)
			if conn != nil {
				enc := gob.NewEncoder(conn)
				initialPack := new(Package)
				initialPack.Circle = circle
				initialPack.KeyStore = keyStore
				err := enc.Encode(initialPack) //send circle to new peer
				if err != nil {
					fmt.Println("initialPack encode error:")
					fmt.Println(err)
				}
			}
		}
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
		fmt.Println("Please input to choose an action: 1 for new transaction, 2 to show ledger, 3 print Uuid ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Input error, try again")
		} else {
			switch input {
			case "1\n":
				{
					fmt.Println("Please input Uuid of receiver:")
					to, err := reader.ReadString('\n')
					fmt.Println("Please input amount:")
					amountString, err := reader.ReadString('\n')
					from := string(x509.MarshalPKCS1PublicKey(myPublicKey))
					to = strings.Replace(to, "\n", "", -1)
					amountString = strings.Replace(amountString, "\n", "", -1)
					amount, err := strconv.Atoi(amountString)
					if err != nil {
						fmt.Println("Error! Amount needs to be an integer")
					} else if amount <= 0 {
						fmt.Println("Error! Amount needs to be positive")
					} else {
						toKey := keyStore.GetKey(to)
						if toKey != "" {
							signedVal := strconv.Itoa(packagesSent) + from + toKey + strconv.Itoa(amount)               //transaction info to be signed
							hashedSVal := sha256.Sum256([]byte(signedVal))                                              //hashed transaction info
							signature, err := rsa.SignPKCS1v15(rand.Reader, myPrivateKey, crypto.SHA256, hashedSVal[:]) //signing hashed transaction info
							if err != nil {
								fmt.Println("Signature creation error:")
								fmt.Println(err)
							}
							transaction := createTransaction(strconv.Itoa(packagesSent), from, toKey, amount, string(signature)) //creating transaction obj
							newPackage := packTransaction(transaction)                                                           //creating package with transaction
							ledger.Transaction(transaction)                                                                      //updating own ledger
							newPackage.Uuid = myID                                                                               //adding uuid to package
							go broadcast(newPackage)                                                                             //broadcasting package to network
							packagesSent++
						} else {
							fmt.Println("Unknown Uuid")
						}

					}
				}
			case "2\n":
				{
					fmt.Println("Current Ledger:")
					ledger.PrintLedger()
				}
			case "3\n":
				{
					fmt.Println("My Uuid:" + myID)
				}
			case "4\n":
				{
					fmt.Println("test:")
					for _, p := range circle.Peers {
						fmt.Println(p)

					}
				}
			case "5\n":
				{
					fmt.Println("dns test")
					for _, x := range dns.m {
						fmt.Println(x.RemoteAddr().String())
					}
				}
			case "6\n":
				{
					fmt.Println("KeyStore test")
					for x := range keyStore.KeyMap {
						fmt.Println(x)
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
		fmt.Println("Keygen error:")
		fmt.Println(err)
	}
	myPublicKey = &myPrivateKey.PublicKey
	tClock = MakeClock()
	gob.Register(Package{}) //registering objects for gob en/decoding
	gob.Register(Circle{})
	gob.Register(SignedTransaction{})
	gob.Register(KeyStore{})
	var reader = bufio.NewReader(os.Stdin) //Create reader to get user input
	fmt.Println("Please input an IP address and port number of known network member")
	address, err := reader.ReadString('\n') //Reads input
	if err != nil {
		fmt.Println("Read input error:")
		fmt.Println(err)
		return
	}
	address = strings.Replace(address, "\n", "", -1)          //Trimming address
	conn, _ := net.DialTimeout("tcp", address, 5*time.Second) //Attempts connection to given address
	name, _ := os.Hostname()                                  //Find own name
	addrs, _ := net.LookupHost(name)                          //Find own address
	ln, _ := net.Listen("tcp", "")                            //Listen for incoming connections
	myAddr = ln.Addr().String()
	for i, a := range addrs { //logic for getting ipv4 address
		chopped := strings.Split(a, ".")
		if len(chopped) == 4 {
			if firstVal, err := strconv.Atoi(chopped[0]); err == nil {
				if firstVal != 127 {
					myAddr = addrs[i] + strings.Replace(myAddr, "[::]", "", -1) //add port to address
				}
			}
		}
	}
	fmt.Println("My address: " + myAddr)
	fmt.Println("my id: " + myID)
	circle.AddPeer(myAddr) //Adds self to Circle
	if conn != nil {
		//address responds
		fmt.Println("got response, joining network")
		dns.AddConnection(conn)
		go handleConnection(conn)
		go listenForConnections(ln)
		joinReq := new(Package)
		enc := gob.NewEncoder(conn)
		joinReq.Address = myAddr
		joinReq.NewComer = true
		joinReq.Key = string(x509.MarshalPKCS1PublicKey(myPublicKey))
		joinReq.Uuid = myID
		err := enc.Encode(joinReq)
		if err != nil {
			fmt.Println("Encode joinReq error:")
			fmt.Println(err)
		}
		takeInputFromUser()
	} else {
		//address not responding
		fmt.Println("no response, starting new network")
		keyStore.AddKey(myID, string(x509.MarshalPKCS1PublicKey(myPublicKey)))
		go listenForConnections(ln)
		takeInputFromUser()
	}

}
