package orgkey

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/artifactcrypto"
	"github.com/ActiveState/cli/internal/constants"
)

// keyServiceHandler serves a valid contract at endpointPath and counts requests.
// When expectToken is non-empty, it requires a matching bearer token.
func keyServiceHandler(t *testing.T, key []byte, org, keyID, expectToken string, fetches *atomic.Int32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fetches.Add(1)
		if r.URL.Path != endpointPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if expectToken != "" && r.Header.Get("Authorization") != "Bearer "+expectToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write(mustJSON(t, contractFields(key, org, keyID)))
	}
}

func TestKeyHappyPathAndSingleFetch(t *testing.T) {
	key := testKey()
	var fetches atomic.Int32
	srv := httptest.NewTLSServer(keyServiceHandler(t, key, "myorg", "kid-1", "secret-token", &fetches))
	defer srv.Close()

	cfg := configForServer(t, srv)
	cfg.strings[constants.PrivateIngredientBearerTokenEnvConfig] = "ORGKEY_TOKEN"
	t.Setenv("ORGKEY_TOKEN", "secret-token")

	p := New(cfg, "myorg")
	defer p.Close()

	gotKey, gotID, err := p.Key(context.Background())
	if err != nil {
		t.Fatalf("Key: %v", err)
	}
	if !bytes.Equal(gotKey, key) {
		t.Error("returned key does not match")
	}
	if gotID != "kid-1" {
		t.Errorf("keyID = %q, want kid-1", gotID)
	}

	// Subsequent calls reuse the in-run cache: still exactly one fetch.
	for i := 0; i < 3; i++ {
		if _, _, err := p.Key(context.Background()); err != nil {
			t.Fatalf("Key (cached): %v", err)
		}
	}
	if got := fetches.Load(); got != 1 {
		t.Errorf("fetch count = %d, want exactly 1", got)
	}
}

func TestKeyBearerTokenFromFile(t *testing.T) {
	key := testKey()
	var fetches atomic.Int32
	srv := httptest.NewTLSServer(keyServiceHandler(t, key, "myorg", "kid", "file-token", &fetches))
	defer srv.Close()

	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("file-token\n"), 0600); err != nil { // trailing newline is trimmed
		t.Fatal(err)
	}
	cfg := configForServer(t, srv)
	cfg.strings[constants.PrivateIngredientBearerTokenFileConfig] = tokenFile

	gotKey, _, err := New(cfg, "myorg").Key(context.Background())
	if err != nil {
		t.Fatalf("Key (bearer file): %v", err)
	}
	if !bytes.Equal(gotKey, key) {
		t.Error("returned key does not match")
	}
}

func TestKeyRefusesNonHTTPS(t *testing.T) {
	cfg := newFakeConfig(t)
	cfg.strings[constants.PrivateIngredientKeyServiceURLConfig] = "http://insecure.example.com"

	_, _, err := New(cfg, "myorg").Key(context.Background())
	if !errors.Is(err, ErrInsecureURL) {
		t.Fatalf("error = %v, want ErrInsecureURL", err)
	}
}

func TestKeyRejectsUntrustedCertificate(t *testing.T) {
	key := testKey()
	var fetches atomic.Int32
	srv := httptest.NewTLSServer(keyServiceHandler(t, key, "myorg", "kid", "", &fetches))
	defer srv.Close()

	// Point at the server but do NOT configure its CA, so its self-signed cert
	// is untrusted.
	cfg := newFakeConfig(t)
	cfg.strings[constants.PrivateIngredientKeyServiceURLConfig] = srv.URL

	if _, _, err := New(cfg, "myorg").Key(context.Background()); err == nil {
		t.Fatal("expected a TLS verification error for an untrusted certificate")
	}
}

func TestKeyFailsClosedOnTimeout(t *testing.T) {
	key := testKey()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, _ = w.Write(mustJSON(t, contractFields(key, "myorg", "kid")))
	}))
	defer srv.Close()
	cfg := configForServer(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if _, _, err := New(cfg, "myorg").Key(ctx); err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
}

func TestKeyMTLS(t *testing.T) {
	key := testKey()
	certPath, keyPath := genClientCert(t)

	var sawClientCert bool
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawClientCert = len(r.TLS.PeerCertificates) > 0
		_, _ = w.Write(mustJSON(t, contractFields(key, "myorg", "kid")))
	}))
	srv.TLS = &tls.Config{ClientAuth: tls.RequireAnyClientCert}
	srv.StartTLS()
	defer srv.Close()

	cfg := configForServer(t, srv)
	cfg.strings[constants.PrivateIngredientMTLSCertConfig] = certPath
	cfg.strings[constants.PrivateIngredientMTLSKeyConfig] = keyPath

	gotKey, _, err := New(cfg, "myorg").Key(context.Background())
	if err != nil {
		t.Fatalf("Key (mTLS): %v", err)
	}
	if !bytes.Equal(gotKey, key) {
		t.Error("returned key does not match")
	}
	if !sawClientCert {
		t.Error("server did not receive a client certificate")
	}
}

func TestKeyRejectsTLSBelow12(t *testing.T) {
	key := testKey()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(mustJSON(t, contractFields(key, "myorg", "kid")))
	}))
	srv.TLS = &tls.Config{MaxVersion: tls.VersionTLS11}
	srv.StartTLS()
	defer srv.Close()
	cfg := configForServer(t, srv)

	if _, _, err := New(cfg, "myorg").Key(context.Background()); err == nil {
		t.Fatal("expected handshake failure against a TLS 1.1 server")
	}
}

func TestNotConfiguredIsNoOp(t *testing.T) {
	cfg := newFakeConfig(t) // no URL set
	p := New(cfg, "myorg")
	if p.Configured() {
		t.Error("Configured() = true with no URL set")
	}
	if _, _, err := p.Key(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("error = %v, want ErrNotConfigured", err)
	}
}

func TestOnDiskCacheReusedAcrossRuns(t *testing.T) {
	key := testKey()
	var fetches atomic.Int32
	srv := httptest.NewTLSServer(keyServiceHandler(t, key, "myorg", "kid", "", &fetches))
	defer srv.Close()

	cfg := configForServer(t, srv)
	cfg.bools[constants.PrivateIngredientCacheKeyConfig] = true

	// First run fetches over the network and writes the cache.
	if _, _, err := New(cfg, "myorg").Key(context.Background()); err != nil {
		t.Fatalf("first Key: %v", err)
	}
	if got := fetches.Load(); got != 1 {
		t.Fatalf("fetch count after first run = %d, want 1", got)
	}

	cachePath := filepath.Join(cfg.dir, cacheFileName)
	info, err := os.Stat(cachePath)
	if err != nil {
		t.Fatalf("cache file not written: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode()&0177 != 0 {
		t.Errorf("cache file mode = %v, want 0600", info.Mode())
	}

	// Second run (fresh provider, same config dir) reads from disk: no new fetch.
	gotKey, _, err := New(cfg, "myorg").Key(context.Background())
	if err != nil {
		t.Fatalf("second Key: %v", err)
	}
	if !bytes.Equal(gotKey, key) {
		t.Error("cached key does not match")
	}
	if got := fetches.Load(); got != 1 {
		t.Errorf("fetch count after cached run = %d, want still 1", got)
	}
}

func TestMemoryOnlyWritesNothingToDisk(t *testing.T) {
	key := testKey()
	var fetches atomic.Int32
	srv := httptest.NewTLSServer(keyServiceHandler(t, key, "myorg", "kid", "", &fetches))
	defer srv.Close()

	cfg := configForServer(t, srv) // cache opt-in left false (default)

	if _, _, err := New(cfg, "myorg").Key(context.Background()); err != nil {
		t.Fatalf("Key: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.dir, cacheFileName)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("cache file should not exist in memory-only mode (stat err = %v)", err)
	}
}

// sanity: artifactcrypto.KeySize is what the contract helpers assume.
func TestKeySizeAssumption(t *testing.T) {
	if artifactcrypto.KeySize != 32 {
		t.Fatalf("unexpected KeySize %d", artifactcrypto.KeySize)
	}
}
