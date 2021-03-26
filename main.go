package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type Block struct {
	Index 			int
	TimeStamp		string
	BPM				int
	Hash 			string
	PrevHash		string
	Validator 		string
}

var Blockchain []Block
var tempBlocks []Block


var candidateBlocks = make(chan Block)

// anouncements broadcasts winning validator to all nodes
var announcements = make(chan string)

var validators = make(map[string]int)

var mutex = &sync.Mutex{}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	t := time.Now()
	genesisBlock := Block{}
	genesisBlock = Block{0, t.String(), 0, calculateBlockHash(genesisBlock), "",
		""}
	Blockchain = append(Blockchain, genesisBlock)

	tcpPort := os.Getenv("PORT")
	server, err := net.Listen("tcp", ":" + tcpPort)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("TCP Server Listening on port:", tcpPort)
	defer server.Close()

	go func() {
		for candidate := range candidateBlocks {
			mutex.Lock()
			tempBlocks = append(tempBlocks, candidate)
			mutex.Unlock()
		}
	}()

	go func() {
		for {
			pickWinner()
		}
	}()

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConn(conn)
	}
}


func pickWinner() {
	time.Sleep(30 * time.Second)
	mutex.Lock()
	temp := tempBlocks
	mutex.Unlock()

	lotteryPool := []string{}
	if len(temp) > 0 {

	OUTER:
		for _, block := range temp {
			for _, node := range lotteryPool {
				if block.Validator == node {
					continue OUTER
				}
			}

			mutex.Lock()
			setValidators := validators
			mutex.Unlock()

			k, ok := setValidators[block.Validator]
			if ok {
				for i := 0; i < k; i++ {
					lotteryPool = append(lotteryPool, block.Validator)
				}
			}
		}

		s := rand.NewSource(time.Now().Unix())
		r := rand.New(s)
		lotteryWinner := lotteryPool[r.Intn(len(lotteryPool))]

		for _, block := range temp {
			if block.Validator == lotteryWinner {
				mutex.Lock()
				Blockchain = append(Blockchain, block)
				mutex.Unlock()
				for _ = range validators {
					announcements <- "\nwinning validator: " + lotteryWinner + "\n"
				}
				break
			}
		}
	}

	mutex.Lock()
	tempBlocks = []Block{}
	mutex.Unlock()
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	go func() {
		for {
			msg := <- announcements
			io.WriteString(conn, msg)
		}
	}()

	var address string
	io.WriteString(conn, "Enter token balance:")
	scanBalance := bufio.NewScanner(conn)
	for scanBalance.Scan() {
		balance, err := strconv.Atoi(scanBalance.Text())
		if err != nil {
			log.Printf("%v not a number: %v", scanBalance.Text(), err)
			return
		}
		t := time.Now()
		address = calculateHash(t.String())
		validators[address] = balance
		fmt.Println(validators)
		break
	}
	log.Println("debug2")

	io.WriteString(conn, "\nEnter a new BPM:")

	scanBPM := bufio.NewScanner(conn)

	go func() {
		for {
			for scanBPM.Scan() {
				bpm, err := strconv.Atoi(scanBPM.Text())
				if err != nil {
					log.Printf("%v not number: %v", scanBPM.Text(), err)
					delete(validators, address)
					conn.Close()
				}

				mutex.Lock()
				oldBlock := Blockchain[len(Blockchain)-1]
				mutex.Unlock()

				newBlock, err := generateBlock(oldBlock, bpm, address)
				if err != nil {
					log.Println(err)
					continue
				}
				if isBlockValid(newBlock, oldBlock) {
					candidateBlocks <- newBlock
				}
				io.WriteString(conn, "\nEnter a new BPM:")
			}
		}
	}()

	for {
		time.Sleep(time.Minute)
		mutex.Lock()
		output, err := json.Marshal(Blockchain)
		mutex.Unlock()
		if err != nil {
			log.Fatal(err)
		}
		io.WriteString(conn, string(output) + "\n")
	}
}

func isBlockValid(newBlock, oldBlock Block) bool {
	if newBlock.Index != oldBlock.Index+1 {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if calculateBlockHash(newBlock) != newBlock.Hash {
		return false
	}
	return true
}

func calculateHash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	hash := h.Sum(nil)
	return hex.EncodeToString(hash)
}

func calculateBlockHash(block Block) string {
	record := string(block.Index) + block.TimeStamp + string(block.BPM) + block.PrevHash
	return calculateHash(record)
}

func generateBlock(oldBlock Block, BPM int, address string) (Block, error) {
	var newBlock Block

	t := time.Now()
	newBlock.Index = oldBlock.Index + 1
	newBlock.TimeStamp = t.String()
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = calculateBlockHash(newBlock)
	newBlock.Validator = address

	return newBlock, nil
}

