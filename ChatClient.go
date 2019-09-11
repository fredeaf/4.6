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

var MessagesLock sync.RWMutex    //En mutex, der bruges til at håndtere MessagesSent
var MessagesSent map[string]bool //Et map, der indeholder en boolsk værdi for alle beskeder
var dns = MakeDNS()

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

func broadcast(outbound chan string) {
	for {
		msg := <-outbound //Læs ny besked
		MessagesLock.RLock()
		if MessagesSent[msg] == false {
			MessagesLock.RUnlock()
			//Beskeden er ikke tidligere sendt
			MessagesLock.Lock()
			dns.lock.Lock()
			for _, x := range dns.m {
				x.Write([]byte(msg)) //Beskeden sendes
			}
			dns.lock.Unlock()
			MessagesSent[msg] = true //Beskeden tilføjes til mængden af sendte beskeder
			fmt.Println(msg)         //printer en besked ud
			MessagesLock.Unlock()
		} else {
			MessagesLock.RUnlock()
		}
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

func takeInputFromUser(outbound chan string) {
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
		outbound <- text
	}
}

func main() {
	MessagesSent = make(map[string]bool)
	outbound := make(chan string)          //channel til ting der skal ud/ind
	var reader = bufio.NewReader(os.Stdin) //ny reader, der læser fra terminalen
	fmt.Println("Please input an IP adress with a port number")
	adress, err := reader.ReadString('\n') //finder den angivede adresse
	if err != nil {
		return
	}
	adress = strings.Replace(adress, "\n", "", -1)           //retter adressen til
	conn, _ := net.DialTimeout("tcp", adress, 5*time.Second) //opretter forbindelse til adressen
	go broadcast(outbound)
	if conn != nil {
		//adressen eksisterer
		defer conn.Close()
		dns.AddConnection(conn)
		go handleConnection(outbound, conn)
		go turnIntoAServer(outbound)
		takeInputFromUser(outbound)
	} else {
		//adressen eksisterer ikke
		go turnIntoAServer(outbound)
		takeInputFromUser(outbound)
	}
}
