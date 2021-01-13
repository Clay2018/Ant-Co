package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	a := 0.12345678 + 0.87654321
	println(a)
	fmt.Println(a)
	conn, err := net.Listen("tcp", "0.0.0.0:8888")
	if err != nil {
		log.Fatal("listen error:", err)

		return
	}

	for {
		log.Println("listening...")
		c, err := conn.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			break
		}
		go handleConn(c)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	defer println("disconnect")
	var c string = "hello"
	conn.Write([]byte(c))
	log.Println(conn.RemoteAddr().String())
	var b []byte
	for {
		a, _ := conn.Read(b)
		if a > 5 {
			break;
		}
		a += a
	}
	println(b)
	log.Println(string(b))
}
