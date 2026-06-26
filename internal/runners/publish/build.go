package publish

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/archiver"
	"github.com/ActiveState/cli/internal/artifactcrypto"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/python/wheel"
	"github.com/ActiveState/cli/internal/runbits/orgkey"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// generateEncryptedArtifact validates the --build inputs, fetches and validates
// the org key, builds the wrapped, encrypted artifact, and points the publish
// flow at it. It fails closed before producing any artifact if the key is
// unavailable, and returns a cleanup function the caller must defer.
func (r *Runner) generateEncryptedArtifact(params *Params) (cleanup func(), rerr error) {
	if params.Filepath != "" {
		return nil, locale.NewInputError("err_publish_build_and_file", "The '[ACTIONABLE]--build[/RESET]' flag cannot be combined with a source archive filepath.")
	}
	if r.project == nil {
		return nil, locale.NewInputError("err_publish_build_no_project", "The '[ACTIONABLE]--build[/RESET]' flag requires a project so the organization can be determined.")
	}

	meta, err := wheel.ResolveMetadata(params.Build, wheel.Metadata{Name: params.Name, Version: params.Version})
	if err != nil {
		return nil, locale.WrapInputError(err, "err_publish_build_metadata", "Could not determine the ingredient name and version: {{.V0}}", errs.JoinMessage(err))
	}

	// Fetch and validate the org key before building anything: a private publish
	// is encrypted-required, so fail closed before any byte could be uploaded.
	provider := orgkey.New(r.cfg, r.project.Owner())
	if !provider.Configured() {
		return nil, locale.NewInputError("err_publish_orgkey_unconfigured", "No organization key service is configured, so this private ingredient cannot be encrypted.")
	}
	defer provider.Close()
	key, keyID, err := provider.Key(context.Background())
	if err != nil {
		return nil, locale.WrapInputError(err, "err_publish_orgkey_unavailable", "Could not obtain the organization key, so nothing was uploaded: {{.V0}}", errs.JoinMessage(err))
	}

	archivePath, cleanup, err := buildWrappedArtifact(params.Build, *meta, key, keyID)
	if err != nil {
		return nil, errs.Wrap(err, "Could not build encrypted artifact")
	}

	// TODO(ENG-1641): once the platform supports the genesis/timeless publish
	// flag, set it on the publish mutation so a private publish never advances
	// any commit's at_time. Until then --build cannot be genesis-stamped.

	params.Filepath = archivePath
	if params.Name == "" {
		params.Name = meta.Name
	}
	if params.Version == "" {
		params.Version = meta.Version
	}
	return cleanup, nil
}

// requireOrgNamespace ensures ns belongs to the project owner's private org, so
// an artifact encrypted under that org's key is published under that same org
// and stays decryptable by its consumers.
func requireOrgNamespace(ns, owner string) error {
	org := model.NewNamespaceOrg(owner, "").String()
	if ns == org || strings.HasPrefix(ns, org+"/") {
		return nil
	}
	return locale.NewInputError("err_publish_build_namespace",
		"The '[ACTIONABLE]--build[/RESET]' flag requires a namespace under '[ACTIONABLE]{{.V0}}[/RESET]'.", org)
}

// payloadInstallDir is the directory inside the wrapped artifact that holds the
// deployable payload; the cleartext runtime.json points the consume side at it.
const payloadInstallDir = "install"

// buildWrappedArtifact packs srcDir into a wheel under the given metadata,
// encrypts it under the org key, and wraps the ciphertext together with a
// cleartext runtime.json into a tar.gz ready for upload. It returns the wrapped
// archive path and a cleanup function the caller must invoke once the upload is
// done.
//
// Only ciphertext plus the cleartext envdef ever reaches the wrapped archive:
// the plaintext wheel and payload are removed before the function returns, so no
// plaintext outlives the build.
func buildWrappedArtifact(srcDir string, meta wheel.Metadata, key []byte, keyID string) (archivePath string, cleanup func(), rerr error) {
	tmpDir, err := os.MkdirTemp("", "state-publish-build-")
	if err != nil {
		return "", nil, errs.Wrap(err, "Could not create temp dir")
	}
	cleanup = func() { _ = os.RemoveAll(tmpDir) }
	defer func() {
		if rerr != nil {
			cleanup()
		}
	}()

	wheelPath, err := wheel.Pack(srcDir, meta, tmpDir)
	if err != nil {
		return "", nil, errs.Wrap(err, "Could not build a wheel from %s", srcDir)
	}

	// Assemble the tar.gz that becomes the encrypted payload, placing the wheel
	// under the install dir the consume side deploys.
	plaintextPayload := filepath.Join(tmpDir, "payload.tar.gz")
	if err := archiver.CreateTgz(plaintextPayload, tmpDir, []archiver.FileMap{
		{Source: wheelPath, Target: path.Join(payloadInstallDir, filepath.Base(wheelPath))},
	}); err != nil {
		return "", nil, errs.Wrap(err, "Could not assemble payload")
	}

	ciphertextPath := filepath.Join(tmpDir, "payload.enc")
	if err := encryptFile(plaintextPayload, ciphertextPath, key, keyID); err != nil {
		return "", nil, errs.Wrap(err, "Could not encrypt payload")
	}

	// Drop the plaintext now that only ciphertext is needed; nothing plaintext
	// survives into the wrapped artifact or beyond this point.
	for _, p := range []string{wheelPath, plaintextPayload} {
		if err := os.Remove(p); err != nil {
			return "", nil, errs.Wrap(err, "Could not remove plaintext")
		}
	}

	runtimeJSONPath := filepath.Join(tmpDir, "runtime.json")
	if err := writeRuntimeJSON(runtimeJSONPath); err != nil {
		return "", nil, errs.Wrap(err, "Could not write runtime.json")
	}

	archivePath = filepath.Join(tmpDir, "ingredient.tar.gz")
	if err := archiver.CreateTgz(archivePath, tmpDir, []archiver.FileMap{
		{Source: ciphertextPath, Target: "payload.enc"},
		{Source: runtimeJSONPath, Target: "runtime.json"},
	}); err != nil {
		return "", nil, errs.Wrap(err, "Could not wrap artifact")
	}

	if sha, err := fileutils.Sha256Hash(archivePath); err == nil {
		logging.Debug("Built private ingredient artifact %s (sha256=%s)", filepath.Base(archivePath), sha)
	}

	return archivePath, cleanup, nil
}

// encryptFile streams srcPath through the content-encryption package into a new
// file at dstPath under the given key.
func encryptFile(srcPath, dstPath string, key []byte, keyID string) (rerr error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return errs.Wrap(err, "Could not open payload")
	}
	defer func() {
		if cerr := src.Close(); cerr != nil {
			rerr = errs.Pack(rerr, errs.Wrap(cerr, "Could not close payload"))
		}
	}()

	dst, err := os.Create(dstPath)
	if err != nil {
		return errs.Wrap(err, "Could not create ciphertext")
	}
	defer func() {
		if cerr := dst.Close(); cerr != nil {
			rerr = errs.Pack(rerr, errs.Wrap(cerr, "Could not close ciphertext"))
		}
	}()

	if err := artifactcrypto.Encrypt(src, dst, key, keyID); err != nil {
		return errs.Wrap(err, "Could not encrypt")
	}
	return nil
}

// writeRuntimeJSON writes the minimal cleartext envdef the consume side reads to
// deploy the decrypted payload.
func writeRuntimeJSON(destPath string) error {
	def := struct {
		Env        []json.RawMessage `json:"env"`
		Transforms []json.RawMessage `json:"file_transforms"`
		InstallDir string            `json:"installdir"`
	}{
		Env:        []json.RawMessage{},
		Transforms: []json.RawMessage{},
		InstallDir: payloadInstallDir,
	}
	b, err := json.Marshal(def)
	if err != nil {
		return errs.Wrap(err, "Could not marshal runtime.json")
	}
	if err := os.WriteFile(destPath, b, 0644); err != nil {
		return errs.Wrap(err, "Could not write runtime.json")
	}
	return nil
}
