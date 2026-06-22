package orgkey

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

const (
	// fetchTimeout bounds a single key-service request.
	fetchTimeout = 15 * time.Second
	// maxResponseBytes caps the contract response read from the key service.
	maxResponseBytes = 1 << 20 // 1 MiB
)

// provider fetches the org key over HTTPS and caches it in memory for the run.
type provider struct {
	cfg   configurable
	owner string

	mu    sync.Mutex
	done  bool
	key   []byte
	keyID string
	err   error
}

// New returns a Provider that reads its key-service configuration from cfg and
// validates the fetched key against owner (the project's organization).
func New(cfg configurable, owner string) Provider {
	return &provider{cfg: cfg, owner: owner}
}

func (p *provider) Configured() bool {
	return p.cfg.GetString(constants.PrivateIngredientKeyServiceURLConfig) != ""
}

// Key fetches and validates the org key on first call and returns the cached
// result (including a cached error) on every subsequent call in the run.
func (p *provider) Key(ctx context.Context) ([]byte, string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done {
		return p.key, p.keyID, p.err
	}
	p.done = true
	p.key, p.keyID, p.err = p.load(ctx)
	return p.key, p.keyID, p.err
}

func (p *provider) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range p.key {
		p.key[i] = 0
	}
	p.key = nil
}

func (p *provider) load(ctx context.Context) (key []byte, keyID string, err error) {
	if !p.Configured() {
		return nil, "", ErrNotConfigured
	}

	if p.diskCacheEnabled() {
		if raw, ok := p.readDiskCache(); ok {
			if key, keyID, err := validateContract(raw, p.owner); err == nil {
				return key, keyID, nil
			} else {
				logging.Warning("Ignoring invalid on-disk org key cache: %v", errs.JoinMessage(err))
			}
		}
	}

	raw, err := p.fetch(ctx)
	if err != nil {
		return nil, "", errs.Wrap(err, "unable to fetch org key")
	}
	key, keyID, err = validateContract(raw, p.owner)
	if err != nil {
		return nil, "", errs.Wrap(err, "unable to validate org key")
	}

	if p.diskCacheEnabled() {
		if werr := p.writeDiskCache(raw); werr != nil {
			logging.Warning("Could not cache org key on disk: %v", errs.JoinMessage(werr))
		}
	}
	return key, keyID, nil
}

// fetch performs the HTTPS GET against the configured key service and returns
// the raw contract body.
func (p *provider) fetch(ctx context.Context) ([]byte, error) {
	base := p.cfg.GetString(constants.PrivateIngredientKeyServiceURLConfig)
	u, err := url.Parse(base)
	if err != nil {
		return nil, errs.Wrap(err, "unable to parse key service URL")
	}
	if u.Scheme != "https" {
		return nil, ErrInsecureURL
	}
	u.Path = strings.TrimRight(u.Path, "/") + endpointPath

	client, err := p.httpClient()
	if err != nil {
		return nil, errs.Wrap(err, "unable to build key service HTTP client")
	}

	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errs.Wrap(err, "unable to build key service request")
	}
	if err := p.applyAuth(req); err != nil {
		return nil, errs.Wrap(err, "unable to apply key service authentication")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errs.Wrap(err, "unable to reach key service")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errs.New("key service returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, errs.Wrap(err, "unable to read key service response")
	}
	return body, nil
}

// httpClient builds an HTTPS client enforcing TLS 1.2+, the configured CA or
// pinned certificate, optional mTLS, and a refusal to follow redirects (the URL
// is pinned).
func (p *provider) httpClient() (*http.Client, error) {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}

	if caPath := p.cfg.GetString(constants.PrivateIngredientKeyServiceCAConfig); caPath != "" {
		pem, err := os.ReadFile(caPath)
		if err != nil {
			return nil, errs.Wrap(err, "unable to read key service CA file")
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, errs.New("key service CA file contains no valid certificates")
		}
		tlsCfg.RootCAs = pool
	}

	certPath := p.cfg.GetString(constants.PrivateIngredientMTLSCertConfig)
	keyPath := p.cfg.GetString(constants.PrivateIngredientMTLSKeyConfig)
	if certPath != "" || keyPath != "" {
		if certPath == "" || keyPath == "" {
			return nil, errs.New("mTLS requires both a client certificate and a client key")
		}
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, errs.Wrap(err, "unable to load mTLS client certificate")
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	return &http.Client{
		Timeout:   fetchTimeout,
		Transport: &http.Transport{TLSClientConfig: tlsCfg},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errs.New("key service redirects are not allowed")
		},
	}, nil
}

// applyAuth attaches a bearer token to the request when one is configured. mTLS
// (if configured) is applied at the transport layer in httpClient.
func (p *provider) applyAuth(req *http.Request) error {
	token, err := p.bearerToken()
	if err != nil {
		return errs.Wrap(err, "unable to read bearer token")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return nil
}

// bearerToken reads the short-lived bearer token from the configured env var or
// file. It never returns the token in an error.
func (p *provider) bearerToken() (string, error) {
	if envName := p.cfg.GetString(constants.PrivateIngredientBearerTokenEnvConfig); envName != "" {
		return strings.TrimSpace(os.Getenv(envName)), nil
	}
	if path := p.cfg.GetString(constants.PrivateIngredientBearerTokenFileConfig); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", errs.Wrap(err, "unable to read bearer token file")
		}
		return strings.TrimSpace(string(b)), nil
	}
	return "", nil
}
