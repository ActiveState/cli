// Package orgkey fetches and validates an organization's single AES-256
// encryption key from the customer-hosted HTTPS key service, caches it for the
// duration of a run, and hands the raw key bytes to the artifactcrypto
// primitives. The key is read only from the customer's own service and is never
// placed in a request to the ActiveState Platform.
//
// The custody backend lives caller-side (it makes network calls and reads
// config).
package orgkey

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/ActiveState/cli/internal/artifactcrypto"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
)

func init() {
	configMediator.RegisterOption(constants.PrivateIngredientKeyServiceURLConfig, configMediator.String, "")
	configMediator.RegisterOption(constants.PrivateIngredientKeyServiceCAConfig, configMediator.String, "")
	configMediator.RegisterOption(constants.PrivateIngredientMTLSCertConfig, configMediator.String, "")
	configMediator.RegisterOption(constants.PrivateIngredientMTLSKeyConfig, configMediator.String, "")
	configMediator.RegisterOption(constants.PrivateIngredientBearerTokenEnvConfig, configMediator.String, "")
	configMediator.RegisterOption(constants.PrivateIngredientBearerTokenFileConfig, configMediator.String, "")
	configMediator.RegisterOption(constants.PrivateIngredientCacheKeyConfig, configMediator.Bool, false)
}

var (
	// ErrNotConfigured indicates no key-service URL has been configured.
	ErrNotConfigured = errs.New("org key service is not configured")
	// ErrInsecureURL indicates the configured key-service URL is not https.
	ErrInsecureURL = errs.New("org key service URL must use https")
	// ErrUnknownSchema indicates the contract's schema field is not recognized.
	ErrUnknownSchema = errs.New("org key contract has an unrecognized schema")
	// ErrOrgMismatch indicates the contract is for a different organization than the project's.
	ErrOrgMismatch = errs.New("org key does not belong to this project's organization")
	// ErrBadAlgorithm indicates the contract specifies an unsupported algorithm.
	ErrBadAlgorithm = errs.New("org key contract specifies an unsupported algorithm")
	// ErrBadEncoding indicates the contract's key encoding is unsupported or the key is not valid base64.
	ErrBadEncoding = errs.New("org key contract specifies an unsupported or invalid key encoding")
	// ErrBadKeyLength indicates the decoded key is not a 32-byte AES-256 key.
	ErrBadKeyLength = errs.New("org key must be 32 bytes (AES-256)")
	// ErrFingerprintMismatch indicates the decoded key does not match its stated fingerprint.
	ErrFingerprintMismatch = errs.New("org key does not match its stated fingerprint")
)

const (
	contractSchema    = "activestate.pim.orgkey/v1"
	contractAlgorithm = "AES-256-GCM"
	contractEncoding  = "base64"
	// endpointPath is appended to the configured base URL to form the request URL.
	endpointPath = "/v1/org-key"
)

// configurable is the subset of the config instance this package reads.
type configurable interface {
	GetString(key string) string
	GetBool(key string) bool
	ConfigPath() string
}

// Provider supplies the organization's AES-256 key for a run. Implementations
// fetch and validate the key on first use and return the cached value
// thereafter; the at-rest backend is swappable behind this interface.
type Provider interface {
	// Configured reports whether a key service has been configured. When it
	// returns false the provider is a no-op and Key returns ErrNotConfigured.
	Configured() bool
	// Key returns the raw 32-byte org key and its id for this run.
	Key(ctx context.Context) (key []byte, keyID string, err error)
	// Close zeroizes any in-memory key material held by the provider.
	Close()
}

// contract is the org-key JSON document served by the key service.
type contract struct {
	Schema      string `json:"schema"`
	Org         string `json:"org"`
	KeyID       string `json:"key_id"`
	Algorithm   string `json:"algorithm"`
	Encoding    string `json:"encoding"`
	Key         string `json:"key"`
	Fingerprint string `json:"fingerprint"`
}

// validateContract parses raw, checks it against the expected organization, and
// returns the decoded 32-byte key and its id. Errors never include the key
// bytes.
func validateContract(raw []byte, expectedOrg string) (key []byte, keyID string, err error) {
	var c contract
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, "", errs.Wrap(err, "unable to parse org key contract")
	}
	if c.Schema != contractSchema {
		return nil, "", ErrUnknownSchema
	}
	if !strings.EqualFold(c.Org, expectedOrg) {
		return nil, "", ErrOrgMismatch
	}
	if c.Algorithm != contractAlgorithm {
		return nil, "", ErrBadAlgorithm
	}
	if c.Encoding != contractEncoding {
		return nil, "", ErrBadEncoding
	}
	key, decErr := base64.StdEncoding.DecodeString(strings.TrimPrefix(c.Key, "b64:"))
	if decErr != nil {
		return nil, "", ErrBadEncoding
	}
	if len(key) != artifactcrypto.KeySize {
		return nil, "", ErrBadKeyLength
	}
	if artifactcrypto.Fingerprint(key) != c.Fingerprint {
		return nil, "", ErrFingerprintMismatch
	}
	return key, c.KeyID, nil
}
