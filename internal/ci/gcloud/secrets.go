package gcloud

import (
	"context"
	"fmt"

	"cloud.google.com/go/compute/metadata"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"

	"github.com/ActiveState/cli/internal/logging"
)

// ErrNotAvailable means gcloud cannot be accessed due to missing env vars
type ErrNotAvailable struct{}

func (e ErrNotAvailable) Error() string { return "" }

// GetSecret accesses the payload for the given secret
func GetSecret(name string) (string, error) {
	// Create the client.
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		// gcloud does not expose the error type for "no credentials", so we're going to assume any error is a not available error
		logging.Debug("Gcloud Secretmanager failed to initialize (ignore if you're not trying to use gcloud): %v", err)
		return "", fmt.Errorf("failed to create gcloud secretmanager client: %v %w", err, ErrNotAvailable{})
	}

	projectID, err := metadata.ProjectID()
	if err != nil {
		// we might not be on gcloud at all, for the time being I've yet to find a cheap way to identify this
		logging.Debug("Gcloud could not get project ID (ignore if you're not trying to use gcloud): %v", err)
		return "", fmt.Errorf("failed to create gcloud metadata client: %v %w", err, ErrNotAvailable{})
	}

	// Formulate the secret path
	path := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, name)
	logging.Debug("Accessing gcloud secret at: %s", path)

	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: path,
	}

	// Call the API.
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %v", err)
	}

	return string(result.Payload.Data), nil
}
