package simpleresty

import (
	"context"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

func OAuth(clientID, clientSecret string, endpoint oauth2.Endpoint) (*oauth2.Token, error) {
	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     endpoint,
	}

	// Redirect user to consent page to ask for permission
	// for the scopes specified above.
	url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	fmt.Printf("Visit the URL for the auth dialog:\n%v", url)

	// Use the authorization code that is pushed to the redirect
	// URL. Exchange will do the handshake to retrieve the
	// initial access token. The HTTP Client returned by
	// conf.Client will refresh the token as necessary.
	var code string
	fmt.Printf("Enter the code in the return URL: ")
	if _, err := fmt.Scan(&code); err != nil {
		return nil, err
	}
	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	return tok, nil
}

func OAuthClientCredentials(clientID, clientSecret, tokenURL string) (*oauth2.Token, error) {
	ctx := context.Background()
	conf := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
	}

	token, retrieveErr := conf.Token(ctx)
	if retrieveErr != nil {
		return nil, retrieveErr
	}

	return token, nil
}
