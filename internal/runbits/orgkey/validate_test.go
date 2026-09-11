package orgkey

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/artifactcrypto"
	"github.com/ActiveState/cli/internal/constants"
)

func TestValidateContract(t *testing.T) {
	key := testKey()

	tests := []struct {
		name    string
		mutate  func(m map[string]string)
		wantErr error // nil means success
	}{
		{name: "valid", mutate: func(map[string]string) {}},
		{name: "unknown schema", mutate: func(m map[string]string) { m["schema"] = "something/v9" }, wantErr: ErrUnknownSchema},
		{name: "org mismatch", mutate: func(m map[string]string) { m["org"] = "someoneelse" }, wantErr: ErrOrgMismatch},
		{name: "bad algorithm", mutate: func(m map[string]string) { m["algorithm"] = "AES-128-GCM" }, wantErr: ErrBadAlgorithm},
		{name: "bad encoding", mutate: func(m map[string]string) { m["encoding"] = "hex" }, wantErr: ErrBadEncoding},
		{name: "invalid base64", mutate: func(m map[string]string) { m["key"] = "b64:!!!notbase64!!!" }, wantErr: ErrBadEncoding},
		{name: "wrong key length", mutate: func(m map[string]string) {
			m["key"] = "b64:" + base64.StdEncoding.EncodeToString([]byte("too short"))
		}, wantErr: ErrBadKeyLength},
		{name: "fingerprint mismatch", mutate: func(m map[string]string) { m["fingerprint"] = "sha256:deadbeef" }, wantErr: ErrFingerprintMismatch},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fields := contractFields(key, "myorg", "kid-1")
			tc.mutate(fields)
			gotKey, gotID, err := validateContract(mustJSON(t, fields), "myorg")

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !bytes.Equal(gotKey, key) {
				t.Error("decoded key does not match")
			}
			if gotID != "kid-1" {
				t.Errorf("keyID = %q, want kid-1", gotID)
			}
		})
	}
}

func TestValidateContractOrgIsCaseInsensitive(t *testing.T) {
	key := testKey()
	fields := contractFields(key, "MyOrg", "kid")
	if _, _, err := validateContract(mustJSON(t, fields), "myorg"); err != nil {
		t.Fatalf("expected case-insensitive org match, got %v", err)
	}
}

func TestValidateContractRejectsGarbageJSON(t *testing.T) {
	if _, _, err := validateContract([]byte("not json"), "myorg"); err == nil {
		t.Fatal("expected an error for non-JSON contract")
	}
}

func TestPreflightKey(t *testing.T) {
	key := testKey()
	other := make([]byte, artifactcrypto.KeySize) // all zeros, different key

	var payload bytes.Buffer
	if err := artifactcrypto.Encrypt(strings.NewReader("private wheel"), &payload, key, "kid"); err != nil {
		t.Fatal(err)
	}

	if err := PreflightKey(bytes.NewReader(payload.Bytes()), key); err != nil {
		t.Errorf("PreflightKey(correct key) = %v, want nil", err)
	}
	if err := PreflightKey(bytes.NewReader(payload.Bytes()), other); !errors.Is(err, artifactcrypto.ErrWrongKey) {
		t.Errorf("PreflightKey(wrong key) = %v, want ErrWrongKey", err)
	}
}

func TestSanitizeChildEnv(t *testing.T) {
	cfg := newFakeConfig(t)
	cfg.strings[constants.PrivateIngredientBearerTokenEnvConfig] = "ORGKEY_TOKEN"

	env := map[string]string{"ORGKEY_TOKEN": "secret", "PATH": "/usr/bin"}
	SanitizeChildEnv(cfg, env)
	if _, ok := env["ORGKEY_TOKEN"]; ok {
		t.Error("bearer-token env var was not scrubbed")
	}
	if env["PATH"] != "/usr/bin" {
		t.Error("unrelated env var was removed")
	}

	// No configured token var: nothing is removed.
	env2 := map[string]string{"PATH": "/usr/bin"}
	SanitizeChildEnv(newFakeConfig(t), env2)
	if len(env2) != 1 {
		t.Error("env modified when no token var configured")
	}
}
