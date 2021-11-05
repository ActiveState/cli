package main

import (
	"fmt"
	"github.com/davidji99/rollrest-go/rollrest"
)

func main() {
	client, newClientErr := rollrest.New(rollrest.AuthAAT("some_account_access_token"),
		rollrest.UserAgent("rollbar-go-custom"))

	if newClientErr != nil {
		fmt.Printf("Error: %v\n", newClientErr)
		return
	}

	fmt.Println(client)
}
