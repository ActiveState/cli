package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("first arg should be patth to socket file")
	}

	path := os.Args[1]

	conn, err := net.Dial("unix", path)
	if err != nil {
		log.Fatalln(err)
	}

	if _, err := conn.Write([]byte("http-addr")); err != nil {
		log.Fatalln(err)
	}

	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(string(buf[:n]))
}
