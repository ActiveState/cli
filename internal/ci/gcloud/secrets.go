package gcloud

import (
	"context"
	"fmt"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
)

// ErrNotAvailable means gcloud cannot be accessed due to missing env vars
type ErrNotAvailable struct{}

func (e ErrNotAvailable) Error() string { return "" }

// GetSecret accesses the payload for the given secret
func GetSecret(name string) (string, error) {
	// Check if we have the project ID set via env var
	var project string
	var found bool
	if project, found = os.LookupEnv(constants.GcloudProjectEnvVarName); ! found {
		return "", fmt.Errorf("%s is required %w", constants.GcloudProjectEnvVarName, ErrNotAvailable{})
	}

	// Formulate the secret path
	path := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", project, name)
	logging.Debug("Accessing gcloud secret at: %s", path)

	// Create the client.
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		// gcloud does not expose the error type for "no credentials", so we're going to assume any error is a not available error
		logging.Debug("%v", err)
		return "", fmt.Errorf("failed to create secretmanager client: %v %w", err, ErrNotAvailable{})
	}

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
