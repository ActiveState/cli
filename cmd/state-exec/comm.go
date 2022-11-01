package main

import (
	"fmt"
	"net"
)

const (
	network  = "unix"
	msgWidth = 1024
)

type svcMsg interface {
	SvcMsg() string
}

func sendMsgToService(sockPath string, m svcMsg) error {
	conn, err := net.Dial(network, sockPath)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(m.SvcMsg()))
	if err != nil {
		return fmt.Errorf("write to connection failed: %w", err)
	}

	buf := make([]byte, msgWidth)
	_, err = conn.Read(buf)
	if err != nil {
		return fmt.Errorf("read from connection failed: %w", err)
	}

	return nil
}
