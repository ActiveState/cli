package flisten

import (
	"fmt"
	"net"
	"time"
)

func noopPrint(a ...interface{})  {}
func fmtPrintln(a ...interface{}) { _, _ = fmt.Println(a...) }

type slowConn struct {
	net.Conn
	output func(a ...interface{})
}

func newSlowConn(c net.Conn, debug bool) *slowConn {
	output := noopPrint
	if debug {
		output = fmtPrintln
	}

	return &slowConn{
		Conn:   c,
		output: output,
	}
}

func (c *slowConn) Read(b []byte) (n int, err error) {
	c.output("slow read start pause")
	time.Sleep(time.Second * 3)

	c.output("slow read reading")
	n, err = c.Conn.Read(b)

	c.output("slow read done pause")
	time.Sleep(time.Second * 3)

	return n, err
}

func (c *slowConn) Write(b []byte) (n int, err error) {
	c.output("slow write start pause")
	time.Sleep(time.Second * 3)

	c.output("slow write writing")
	n, err = c.Conn.Write(b)

	c.output("slow write done pause")
	time.Sleep(time.Second * 3)

	return n, err
}
