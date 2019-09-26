package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/gqlclient"
)

func main() {
	var (
		isTest bool
		filter string
	)

	flag.BoolVar(&isTest, "test", isTest, "use mocks")
	flag.StringVar(&filter, "filter", filter, "output filtered users")
	flag.Parse()

	gqlEndpoint := "https://api.graph.cool/simple/v1/ciyz901en4j590185wkmexyex"
	gqlHdrs := gqlclient.Header{
		"Cache-Control": {"no-cache"},
	}
	prv := newProvider(isTest, gqlEndpoint, gqlHdrs) // factory

	if err := work(prv, filter); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}

func work(prv provider, filter string) error {
	if filter != "" {
		resp, err := prv.allUsersByNameContains(filter)
		if err != nil {
			return err
		}

		msg := fmt.Sprintf("## printing users with %q in name", filter)
		return printResp(msg, resp)
	}

	resp, err := prv.allUsers()
	if err != nil {
		return err
	}

	return printResp("## printing all users:", resp)
}

func printResp(msg string, resp *allUsersResp) error {
	if msg == "" {
		return errors.New("no message provided")
	}

	fmt.Println(msg)

	if len(resp.AllUsers) == 0 {
		fmt.Println("no users to print")
		return nil
	}

	for _, u := range resp.AllUsers {
		fmt.Println(u)
	}

	return nil
}
