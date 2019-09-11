package __6

import (
	"bufio"
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
var tClock clock

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
func MakeClock(dns DNS) *clock {
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

func broadcast(transaction Transaction) {
	ledger.lock.Lock()
	defer ledger.lock.Unlock()
	if checkClock(transaction.ID) { //checks for updated
		dns.lock.Lock()
		defer dns.lock.Unlock()
		for _, x := range dns.m {
			x.Write((transaction)) //new transaction is propagated
		}
		fmt.Println(ledger) //Updated ledger is printed
	} else {
	}
}

func handleConnection(outbound chan string, conn net.Conn) {
	defer dns.RemoveConnection(conn)
	defer conn.Close()
	for {
		otherEnd := conn.RemoteAddr().String()
		for {
			msg, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				fmt.Println("Ending session with " + otherEnd)
				return
			}
			outbound <- msg
		}
	}
}

func turnIntoAServer(outbound chan string) {
	name, _ := os.Hostname()         //Finder eget navn
	addrs, _ := net.LookupHost(name) //Finder egen adresse
	for indx, addr := range addrs {
		//Printer adresse
		fmt.Println("Address number " + strconv.Itoa(indx) + ": " + addr)
	}
	ln, _ := net.Listen("tcp", "") //lytter efter forbindelse udefra
	defer ln.Close()
	for {
		_, port, _ := net.SplitHostPort(ln.Addr().String()) //Finder den port der lyttes på
		fmt.Println("Listening on port " + port)
		conn, _ := ln.Accept() //accepterer indgående TCP-connections
		dns.AddConnection(conn)
		go handleConnection(outbound, conn)
	}
}

func createTransaction(todo string) Transaction { //TODO translate incoming/outgoing stuff
	transaction := Transaction{"", "", "", 2}
	return transaction
}

func takeInputFromUser() {
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("> ")
		text, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		if text == "quit\n" {
			return
		}
		go broadcast(createTransaction(text)) //TODO handle incoming stuff
	}
}

func main() {
	outbound := make(chan string)          //channel til ting der skal ud/ind
	var reader = bufio.NewReader(os.Stdin) //ny reader, der læser fra terminalen
	fmt.Println("Please input an IP adress with a port number")
	adress, err := reader.ReadString('\n') //finder den angivede adresse
	if err != nil {
		return
	}
	adress = strings.Replace(adress, "\n", "", -1)           //retter adressen til
	conn, _ := net.DialTimeout("tcp", adress, 5*time.Second) //opretter forbindelse til adressen
	if conn != nil {
		//adressen eksisterer
		defer conn.Close()
		dns.AddConnection(conn)
		go handleConnection(outbound, conn)
		go turnIntoAServer(outbound)
		takeInputFromUser()
		tClock = MakeClock(dns)
	} else {
		//adressen eksisterer ikke
		go turnIntoAServer(outbound)
		takeInputFromUser()
		tClock = MakeClock(dns)
	}
}
