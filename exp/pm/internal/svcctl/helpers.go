package svcctl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ActiveState/cli/exp/pm/internal/ipc"
	"github.com/ActiveState/cli/exp/pm/internal/svccomm"
)

// TODO: relocate? maybe move into structured type field within svcctl type

func EnsureAndLocateHTTP(n *ipc.Namespace) (addr string, err error) {
	ipcClient := ipc.NewClient(n)
	emsg := "ensure svc and locate http: %w"
	commClient := svccomm.NewClient(ipcClient)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*2)
	defer cancel()

	addr, err = commClient.GetHTTPAddr(ctx)
	if err != nil {
		var sderr *ipc.ServerDownError
		if !errors.As(err, &sderr) {
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

func LocateHTTP(n *ipc.Namespace) (addr string, err error) {
	ipcClient := ipc.NewClient(n)
	emsg := "locate http: %w"
	commClient := svccomm.NewClient(ipcClient)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*2)
	defer cancel()

	addr, err = commClient.GetHTTPAddr(ctx)
	fmt.Println(addr)
	if err != nil {
		return "", fmt.Errorf(emsg, err)
	}

	return addr, nil
}

func StopServer(n *ipc.Namespace) (err error) {
	ipcClient := ipc.NewClient(n)
	emsg := "locate http: %w"

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*2)
	defer cancel()

	svcCtl := New(ipcClient)
	if err := svcCtl.Stop(ctx); err != nil {
		return fmt.Errorf(emsg, err)
	}
	return nil
}
