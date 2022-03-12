package svcctl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ActiveState/cli/exp/pm/internal/ipc"
	"github.com/ActiveState/cli/exp/pm/internal/svccomm"
)

func EnsureAndLocateHTTP(n *ipc.Namespace) (addr string, err error) {
	ipcClient := ipc.NewClient(n)
	emsg := "ensure svc and locate http: %w"
	commClient := svccomm.NewClient(ipcClient)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*2)
	defer cancel()

	addr, err = commClient.GetHTTPAddr(ctx)
	if err != nil {
		if !errors.Is(err, ipc.ErrServerDown) {
			return "", fmt.Errorf(emsg, err)
		}

		fmt.Println("starting service")
		ctx1, cancel1 := context.WithTimeout(context.Background(), time.Millisecond*2)
		defer cancel1()

		svcCtl := New(ipcClient)
		if err := svcCtl.Start(ctx1); err != nil {
			return "", fmt.Errorf(emsg, err)
		}

		ctx2, cancel2 := context.WithTimeout(context.Background(), time.Millisecond*2)
		defer cancel2()
		addr, err = commClient.GetHTTPAddr(ctx2)
		if err != nil {
			return "", fmt.Errorf(emsg, err)
		}
	}

	return addr, nil
}
