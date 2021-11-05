# rollrest-go
rollrest-go is a Go client library for accessing the [Rollbar REST APIs](https://explorer.docs.rollbar.com/).

# Example
```go
	client, newClientErr := rollrest.New(rollbar.AuthAAT("some_account_access_token"),
		rollbar.UserAgent("rollbar-go-custom"))

	if newClientErr != nil {
		fmt.Printf("Error: %v\n", newClientErr)
		return
	}

	fmt.Println(client)
```