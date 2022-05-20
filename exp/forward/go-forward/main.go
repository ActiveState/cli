package main

import (
	"log"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("first arg should be path to socket file")
	}

	path := os.Args[1]

	conn, err := net.Dial("unix", path)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("http-addr")); err != nil {
		log.Fatalln(err)
	}

	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		log.Fatalln(err)
	}

	_ = n
	//fmt.Println(string(buf[:n]))
}
