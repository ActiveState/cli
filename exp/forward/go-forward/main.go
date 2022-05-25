package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/ActiveState/cli/internal/exeutils"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("first arg should be a path to socket file")
	}
	if len(os.Args) < 3 {
		log.Fatalln("second arg should be a path to a language runtime")
	}

	path := os.Args[1]
	runt := os.Args[2]

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

	code, _, err := exeutils.ExecuteAndPipeStd(runt, os.Args[3:], os.Environ())
	if err != nil {
		fmt.Println(err)
		os.Exit(42)
	}
	os.Exit(code)
}
