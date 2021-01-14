/*
- 创建自己的区块链
- 理解hash函数是如何保持区块链的完整性
- 如何创造并添加新的块
- 多个节点如何竞争生成块
- 通过浏览器来查看整个链
 */

/*
- 接收TCP连接，接收新区块数据
- 广播区块
 */
/*
- 寻求3个端口，建立连接并接收数据
- 本身接收从终端输入
- 接收的广播数据存放至文件
- 产生新区块立马广播

- 一个连接发送和拉取块数据
- 一个连接读取数据，生成新块
 */
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Block struct {
	Index		int
	Timestamp	string
	BPM			int
	Hash		string
	PrevHash	string
}

var Blockchain []Block
var bcServer chan []Block

func calculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hash := h.Sum(nil)
	return hex.EncodeToString(hash)
}

func generateBlock(oldBlock Block, BPM int) (Block, error) {
	var newBlock Block
	t := time.Now()
	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = calculateHash(newBlock)

	return newBlock, nil
}

func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index + 1 != newBlock.Index {
		return false
	}
	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}
	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}
	return true
}

func replaceChain(newBlocks []Block) {
	if len(newBlocks) > len(Blockchain) {
		Blockchain = newBlocks
	}
}

func run() error {
	mux := makeMuxRouter()
	httpAddr := os.Getenv("ADDR")
	log.Println("Listening on ", os.Getenv("ADDR"))
	s := &http.Server {
		Addr: 			":" + httpAddr,
		Handler: 		mux,
		ReadTimeout: 	10 * time.Second,
		WriteTimeout: 	10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func makeMuxRouter() http.Handler {
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handlerWriteBlock).Methods("POST")

	return muxRouter
}

func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(Blockchain, "", " ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	io.WriteString(w, string(bytes))
}

type Message struct {
	BPM int
}

func handlerWriteBlock(w http.ResponseWriter, r *http.Request) {
	var m Message

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&m); err != nil {
		respondWithJSON(w, r, http.StatusBadRequest, r.Body)
	}
	defer r.Body.Close()

	newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], m.BPM)
	if err != nil {
		respondWithJSON(w, r, http.StatusInternalServerError, m)
		return
	}

	if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
		newBlockchain := append(Blockchain, newBlock)
		replaceChain(newBlockchain)
	}

	respondWithJSON(w, r, http.StatusCreated, newBlock)
}

func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
	response, err := json.MarshalIndent(payload, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: internal Server Error"))
		return
	}

	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	bcServer = make(chan[]Block)

	t := time.Now()
	genesisBlock := Block{0, t.String(), 0, "", "00000"}
	hash := calculateHash(genesisBlock)
	genesisBlock.Hash = hash
	Blockchain = append(Blockchain, genesisBlock)

	tcpPort := os.Getenv("PORT")

	server, err := net.Listen("tcp", ":"+ tcpPort)
	log.Println("TCP Server Listening on port: ", tcpPort)
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()
	go func() {
		//建立客户端连接
		time.Sleep(10*time.Second)
		address := "0.0.0.0:8081"
		if tcpPort == "8081" {
			address = "0.0.0.0:8080"
		}
		conn, err := net.Dial("tcp", address)
		if err != nil {
			log.Println(err.Error())
			return
		}
		log.Println("connect conn")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			bb := scanner.Text()
			log.Println("---------begin generate-----------------")
			log.Println(bb[:len(bb)])

			if (len(bb) < 50) {
				log.Println("--------------------------input bpm-------")
				var bpm int
				fmt.Scanln(&bpm)
				io.WriteString(conn, strconv.Itoa(bpm)+"\n")
				continue
	 		} else {
	 			var blockchain []Block
	 			_ = json.Unmarshal([]byte(bb), blockchain)
	 			if len(blockchain) > len(Blockchain) {
	 				replaceChain(blockchain)
				}
	 			continue
			}
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

func handleConn(conn net.Conn) {
	log.Println("begin deal with new connection")
	defer conn.Close()
	io.WriteString(conn, "Enter a new BPM:")
	scanner := bufio.NewScanner(conn)

	go func() {
		for scanner.Scan() {
			tmp := scanner.Text()
			log.Println("----------------")
			log.Println(tmp)
			log.Println("----------------")
			tmp = tmp[:len(tmp)]
			bpm, err := strconv.Atoi(tmp)
			if err != nil {
				log.Printf("%v not a number: %v", scanner.Text(), err)
				continue
			}

			newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], bpm)
			if err != nil {
				log.Println(err)
				continue
			}
			if isBlockValid(newBlock, Blockchain[(len(Blockchain)-1)]) {
				newBlockchain := append(Blockchain, newBlock)
				replaceChain(newBlockchain)
			}

			bcServer <- Blockchain
			io.WriteString(conn, "\nEnter a new BPM:")
		}
	}()

	go func() {
		for {
			time.Sleep(30 * time.Second)
			output, err := json.Marshal(Blockchain)
			if err != nil {
				log.Fatal(err)
			}
			io.WriteString(conn, string(output))
		}
	}()

	for _ = range bcServer {
		log.Println("-------bbbbbbbbbb-----------------------")
	}
}

/*

 */
