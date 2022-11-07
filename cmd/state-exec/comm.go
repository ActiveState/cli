package main

import (
	"fmt"
	"net"

	"github.com/ActiveState/cli/internal/svcctl/svcmsg"
)

const (
	network  = "unix"
	msgWidth = 1024
)

func sendMsgToService(sockPath string, hb *svcmsg.Heartbeat) error {
	conn, err := net.Dial(network, sockPath)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(hb.SvcMsg()))
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
