package main

import (
	"context"
	"fmt"

	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/svcmsg"
)

func sendMsgToService(sockPath *ipc.SockPath, hb *svcmsg.Heartbeat) error {
	client := ipc.NewClient(sockPath)
	_, err := client.Request(context.Background(), hb.SvcMsg())
	if err != nil {
		return fmt.Errorf("send message to service: %w", err)
	}
	return nil
}
