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
	ef := "send msg to service: %w"

	conn, err := net.Dial(network, sockPath)
	if err != nil {
		return fmt.Errorf(ef, err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(hb.SvcMsg()))
	if err != nil {
		return fmt.Errorf(ef, err)
	}

	buf := make([]byte, msgWidth)
	_, err = conn.Read(buf)
	if err != nil {
		return fmt.Errorf(ef, err)
	}

	return nil
}
