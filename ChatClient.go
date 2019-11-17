package main

import (
	"bufio"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
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
var sequencer Sequencer
var blocks []string
var processingTransactions []SignedTransaction
var amount int
var blockRecieved bool
var blockNumber int
var slotLength = 1 //in seconds

type Sequencer struct {
	IP        string
	PublicKey string
}

/*Block skal indenholde:
slot
TransactionList
Draw: Sign(LOTTERY,seed,slot)
VK/PK key fra vinder
predecessor: Hash
signature (injective encoded(json))
*/

type Block struct {
	Slot            int
	TransactionList []string
	Draw            []byte
	Key             string
	Predecessor     [32]byte
	Signature       string
}

type GenBlock struct {
	Seed          string
	Hardness      int
	initialLedger Ledger
	StartTime     time.Time
}

type BlockTree struct {
	transactions []string
	GB           GenBlock
	blockMap     map[[32]byte]Block
	Queue        []Block
}

var GenesisBlock = GenBlock{
	Seed:          "RandomSeed",
	Hardness:      42,
	initialLedger: generateInitialLedger(),
	StartTime:     time.Now(), //TODO: set time??
}

var BT = BlockTree{
	transactions: nil,
	GB:           GenesisBlock,
	blockMap:     make(map[[32]byte]Block),
	Queue:        nil,
}

func generateInitialLedger() Ledger {
	var ledger Ledger
	file, _ := os.Create("KeyFile")
	for i := 0; i < 10; i++ {
		var privatekey, _ = rsa.GenerateKey(rand.Reader, 3000)
		_, _ = io.WriteString(file, string(x509.MarshalPKCS1PrivateKey(privatekey))+",")
		ledger.Accounts[string(x509.MarshalPKCS1PublicKey(&privatekey.PublicKey))] = 10 ^ 6
	}
	return ledger
}

func draw(slot int) []byte {
	signedVal := "LOTTERY" + GenesisBlock.Seed + strconv.Itoa(slot) //TODO: injective encode
	hashedSVal := sha256.Sum256([]byte(signedVal))
	draw, _ := rsa.SignPKCS1v15(rand.Reader, myPrivateKey, crypto.SHA256, hashedSVal[:])
	return draw
}

func checkDraw(draw []byte) bool {
	for i := 0; i < GenesisBlock.Hardness; i++ {
		if draw[i] != 0 {
			return false
		}
	}
	return true
}

func calcSlot(start time.Time) int {
	seconds := int(time.Since(start) / time.Second)
	return seconds / slotLength
}

func getLongestChainRef() [32]byte {
	var bestLength = 0
	var ref [32]byte
	for a, block := range BT.blockMap {
		if block.Slot > bestLength {
			ref = a
		}
	}
	return ref
}

func createNewBlock() {
	var block Block
	block.Slot = calcSlot(GenesisBlock.StartTime)
	if len(BT.blockMap) == 0 {
		block.Predecessor = sha256.Sum256([]byte(GenesisBlock.Seed))
	} else {
		block.Predecessor = getLongestChainRef()
	}
	block.TransactionList = nil //TODO: create transactionList
	block.Draw = draw(block.Slot)
	block.Key = string(x509.MarshalPKCS1PublicKey(myPublicKey))
	signedVal := strconv.Itoa(block.Slot) + string(block.Draw) + block.Key //TODO: injective encode with transactionList and predecessor added
	hashedVal := sha256.Sum256([]byte(signedVal))
	sig, err := rsa.SignPKCS1v15(rand.Reader, myPrivateKey, crypto.SHA256, hashedVal[:])
	if err != nil {
		fmt.Println("error signing new block:")
		fmt.Println(err)
	}
	block.Signature = string(sig)
	if checkDraw(block.Draw) {
		BT.blockMap[hashedVal] = block
		//TODO: send valid block
	}
}

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

func checkTransList(transactions []string) bool {
	//TODO:check transactions, return false if not valid
	return true
}

//Interpret : function for checking the contents of received packages
func interpret(pack *Package) {
	if pack.Transaction != nil {
		if pack.UUID != myID {
			if pack.Transaction.Amount > 0 {
				tClock.lock.Lock()
				defer tClock.lock.Unlock()
				if checkClock(pack.Transaction.ID, pack.UUID) { //checks if transaction is new
					setClock(pack.Transaction.ID, pack.UUID)
					processingTransactions = append(processingTransactions, *pack.Transaction)
					//ledger.Transaction(pack.Transaction) // Ledger is updated with new transaction
					//fmt.Println("Ledger updated: ")
					//ledger.PrintLedger() //Updated ledger is printed
					fmt.Println("Please input to choose an action: 1 for new transaction, 2 to show ledger, 3 print Uuid ")
					broadcast(pack) //Package is sent onward
				}
			}
		}
	}
	if pack.Sequencer.IP != "" {
		sequencer.IP = pack.Sequencer.IP
		sequencer.PublicKey = pack.Sequencer.PublicKey
	}
	if pack.Block.Signature != "" {
		if pack.Block.Slot <= calcSlot(GenesisBlock.StartTime) {
			var pubKey, _ = x509.ParsePKCS1PublicKey([]byte(pack.Block.Key))
			signedVal := "LOTTERY" + GenesisBlock.Seed + strconv.Itoa(pack.Block.Slot) //TODO: injective encode
			var hashedVal = sha256.Sum256([]byte(signedVal))
			err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashedVal[:], []byte(pack.Block.Signature))
			if err != nil {
				fmt.Println("error verifying block:")
				fmt.Println(err)
			} else {
				if checkDraw([]byte(pack.Block.Draw)) {
					//handle block
					if len(pack.Block.TransactionList) > 0 {
						if checkTransList(pack.Block.TransactionList) {
							GenesisBlock.initialLedger.Accounts[pack.Block.Key] += 10 +
								len(pack.Block.TransactionList)
							//add block to tree ?
						}
					} else {
						GenesisBlock.initialLedger.Accounts[pack.Block.Key] += 10
					}
				}
			}
		}
	}

	/*
				if blockVerify(*pack) == true {
					if blockRecieved == false {
		|				blockNumber = pack.Block.ID
						blocks = append(blocks, pack.Block.TransactionID[:]...)
						blockRecieved = true
					} else {
						if blockNumber == pack.Block.ID {
							blockNumber = blockNumber + 1
							for {
								blocks = append(blocks, pack.Block.TransactionID[:]...)
							}
						} else {
							time.Sleep(time.Second)
						}
					}
				}
			}*/
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
		keyStore.AddKey(pack.UUID, pack.Key)
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
							newPackage.UUID = myID                                                                               //adding uuid to package
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
		joinReq.UUID = myID
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

func sendSequencer() {
	pack := packSequencer(&sequencer)
	broadcast(pack)
}

func createSequencer() {
	sequencerPublicKey := Generate("sequencerfileforkey", "SequencersKodeordSomIkkeSkalVære32bytes")
	sequencer.IP = myAddr
	sequencer.PublicKey = sequencerPublicKey
}

func runSequencer() {
	c := 0
	for {
		time.Sleep(10 * time.Second)
		var blocks []string
		var p Package
		for _, transaction := range processingTransactions {
			blocks = append(blocks, transaction.ID)
		}
		p.Block.ID = c
		p.Block.TransactionID = blocks

		signing := strconv.Itoa(c) + "" + strings.Join(blocks, "")
		p.Block.Signature = Sign("sequencerfileforkey", "SequencersKodeordSomIkkeSkalVære32bytes", []byte(signing))
		c = c + 1
		broadcast(&p)
	}
}

func receivingTransactions() {
	for {
		a := blocks[0]
		for t, transaction := range processingTransactions {
			if a == transaction.ID {
				ledger.Transaction(&transaction)
				blocks = blocks[1:]
				processingTransactions = append(processingTransactions[:t], processingTransactions[t+1:]...)
			} else {
				time.Sleep(time.Second)
			}
		}
	}
}

func blockVerify(pack Package) bool {
	packToString := strconv.Itoa(pack.Block.ID) + strings.Join(pack.Block.TransactionID, "")
	stringToBigInt := new(big.Int).SetBytes([]byte(packToString))
	h := sha256.Sum256(stringToBigInt.Bytes())
	x := h[:]
	hashToBigInt := big.NewInt(0).SetBytes(x)
	bigIntToString := hashToBigInt.String()

	neString := strings.Split(sequencer.PublicKey, ";")
	n, _ := new(big.Int).SetString(neString[0], 10)
	e, _ := new(big.Int).SetString(neString[1], 10)

	signatureBig, _ := new(big.Int).SetString(pack.Block.Signature, 10)
	decryptSignature := decrypt(signatureBig, e, n).String()
	if decryptSignature == bigIntToString {
		return true
	}
	return false
}

// Encrypting a plaintext, with the given key and iv.
// After encrypting the text it writes the encryption to Encryptedfile.enc
// which is a new file who just got created. After writing to the file,
// the function prints out a print statement.
func EncryptToFile(key string, plaintext string, fileName string) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		fmt.Println(err)
	}
	cipherText := make([]byte, aes.BlockSize+len(plaintext))
	iv := cipherText[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	fmt.Println("iv,block")
	fmt.Println(len(iv))
	fmt.Println(block.BlockSize())
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], []byte(plaintext))

	err = ioutil.WriteFile(fmt.Sprintf(fileName), cipherText, 0644)
	if err != nil {
		log.Fatalf("Writing encryption file: %s", err)
	} else {
		fmt.Printf("Message encrypted in file: %s\n\n", fileName)
	}
}

// Decrypting the file created by encrypt, with the given key and iv.
// After loading the encrypted file, the function decrypts the file,
// and returns the deciphered text

func DecryptFromFile(key string, fileName string) []byte {
	cipherText, _ := ioutil.ReadFile(fmt.Sprintf(fileName))
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		fmt.Println(err)
	}
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	mode := cipher.NewCTR(block, iv)
	mode.XORKeyStream(cipherText, cipherText)

	return cipherText
}

func KeyGen(keyLength int) (n *big.Int, e *big.Int, d *big.Int) {
	for {
		n = new(big.Int)
		e = big.NewInt(3)
		d = new(big.Int)
		if keyLength <= 3 {
			fmt.Println("Error: key length needs to be at least 4")
			return
		}

		p, err := rand.Prime(rand.Reader, keyLength/2)
		if err != nil {
			fmt.Println("error creating primes:")
			fmt.Println(err)
		}
		q, err := rand.Prime(rand.Reader, keyLength/2)
		if err != nil {
			fmt.Println("error creating primes:")
			fmt.Println(err)
		}

		//calculate n
		n.Set(p)
		n.Mul(n, q)
		if n.BitLen() != keyLength {
			continue
		}

		//calculate d
		q.Sub(q, big.NewInt(1))
		p.Sub(p, big.NewInt(1))

		product := new(big.Int)
		product = product.Mul(q, p)
		//d.Mod(temp, product)

		d = d.ModInverse(e, product)

		if d == nil {
			continue
		}
		//return
		return n, e, d
	}

}

func encrypt(m *big.Int, e *big.Int, n *big.Int) *big.Int {
	c := new(big.Int)
	//m^e mod n
	c = c.Exp(m, e, n)
	return c
}

func decrypt(c *big.Int, d *big.Int, n *big.Int) *big.Int {
	m := new(big.Int)
	//c^d mod n
	m = m.Exp(c, d, n)
	return m
}

func sign(d *big.Int, n *big.Int, m *big.Int) *big.Int {
	h := sha256.New()
	//h(m)
	h.Write(m.Bytes())
	b := new(big.Int)
	b.SetBytes(h.Sum(nil))
	//h(m)^d mod n
	signature := b.Exp(b, d, n)
	return signature
}

func verify(e *big.Int, n *big.Int, m *big.Int, s *big.Int) bool {
	b := new(big.Int)
	res := new(big.Int)
	//s^e mod n
	res.Exp(s, e, n)

	h := sha256.New()
	//h(m)
	h.Write(m.Bytes())
	b.SetBytes(h.Sum(nil))
	fmt.Println("res and s:")
	fmt.Println(res)
	fmt.Println(b)
	if b.Cmp(res) == 0 {
		return true
	}
	return false
}

// Generates a secret and public key, returns the public key and saves the secret key
// in a encrypted file.
func Generate(filename string, password string) string {
	n, e, d := KeyGen(2050)
	publicKey := n.String() + ";" + d.String()
	secretKey := n.String() + ";" + e.String()
	EncryptToFile(password, secretKey, filename)
	return publicKey
}

// Decrypts the message and signs it, returns the signature.
func Sign(filename string, password string, msg []byte) string {
	secretKey := string(DecryptFromFile(password, filename))
	ndString := strings.Split(secretKey, ";")
	n, _ := new(big.Int).SetString(ndString[0], 10)
	d, _ := new(big.Int).SetString(ndString[1], 10)
	msgBI := new(big.Int).SetBytes(msg)
	signature := sign(d, n, msgBI).String()
	return signature
}
