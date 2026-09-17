package orgkey

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/artifactcrypto"
	"github.com/ActiveState/cli/internal/constants"
)

// testKey is a fixed 32-byte AES-256 key used across tests.
func testKey() []byte {
	k := make([]byte, artifactcrypto.KeySize)
	for i := range k {
		k[i] = byte(i + 1)
	}
	return k
}

// fakeConfig is an in-memory implementation of configurable.
type fakeConfig struct {
	strings map[string]string
	bools   map[string]bool
	dir     string
}

func newFakeConfig(t *testing.T) *fakeConfig {
	return &fakeConfig{
		strings: map[string]string{},
		bools:   map[string]bool{},
		dir:     t.TempDir(),
	}
}

func (f *fakeConfig) GetString(key string) string { return f.strings[key] }
func (f *fakeConfig) GetBool(key string) bool     { return f.bools[key] }
func (f *fakeConfig) ConfigPath() string          { return f.dir }

// contractFields returns a valid contract as a field map, so tests can mutate
// individual fields before marshaling.
func contractFields(key []byte, org, keyID string) map[string]string {
	return map[string]string{
		"schema":      contractSchema,
		"org":         org,
		"key_id":      keyID,
		"algorithm":   contractAlgorithm,
		"encoding":    contractEncoding,
		"key":         "b64:" + base64.StdEncoding.EncodeToString(key),
		"fingerprint": artifactcrypto.Fingerprint(key),
	}
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// writeServerCA writes the test server's certificate to a temp PEM file and
// returns its path, for use as the configured key-service CA.
func writeServerCA(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ca.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(path, pemBytes, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// genClientCert generates a self-signed client certificate and key, writes them
// to temp files, and returns their paths (for the mTLS path).
func genClientCert(t *testing.T) (certPath, keyPath string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-client"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	certPath = filepath.Join(dir, "client.crt")
	keyPath = filepath.Join(dir, "client.key")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(certPath, certPEM, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		t.Fatal(err)
	}
	return certPath, keyPath
}

// configForServer returns a fakeConfig pointed at srv with its CA trusted.
func configForServer(t *testing.T, srv *httptest.Server) *fakeConfig {
	cfg := newFakeConfig(t)
	cfg.strings[constants.PrivateIngredientKeyServiceURLConfig] = srv.URL
	cfg.strings[constants.PrivateIngredientKeyServiceCAConfig] = writeServerCA(t, srv)
	return cfg
}
