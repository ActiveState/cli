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

func sendMsgToService(sockPath string, msg svcmsg.Messager) error {
	conn, err := net.Dial(network, sockPath)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(msg.SvcMsg()))
	if err != nil {
		return fmt.Errorf("write to connection failed: %w", err)
	}

	buf := make([]byte, msgWidth)
	_, err = conn.Read(buf)
	if err != nil {
		return fmt.Errorf("read from connection failed (buffer: %q): %w", string(buf), err)
	}

	return nil
}
