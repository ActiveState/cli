package publish

import (
	"cmp"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ActiveState/cli/internal/archiver"
	"github.com/ActiveState/cli/internal/artifactcrypto"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/python/wheel"
	"github.com/ActiveState/cli/internal/runbits/orgkey"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// generateEncryptedArtifact validates the --build inputs, fetches and validates
// the org key, builds the wrapped, encrypted artifact, and points the publish
// flow at it. It fails closed before producing any artifact if the key is
// unavailable, and returns the resolved wheel metadata and a cleanup function
// the caller must defer. Flags take precedence over the meta file, which takes
// precedence over pyproject.toml.
func (r *Runner) generateEncryptedArtifact(params *Params, metaFile request.PublishVariables) (_ *wheel.Metadata, cleanup func(), rerr error) {
	if params.Filepath != "" {
		return nil, nil, locale.NewInputError("err_publish_build_and_file", "The '[ACTIONABLE]--build[/RESET]' flag cannot be combined with a source archive filepath.")
	}
	if r.project == nil {
		return nil, nil, locale.NewInputError("err_publish_build_no_project", "The '[ACTIONABLE]--build[/RESET]' flag requires a project so the organization can be determined.")
	}
	if !fileutils.DirExists(params.Build) {
		return nil, nil, locale.NewInputError("err_publish_build_dir_not_found", "The '[ACTIONABLE]--build[/RESET]' source directory does not exist: [ACTIONABLE]{{.V0}}[/RESET]", params.Build)
	}

	meta, err := wheel.ResolveMetadata(params.Build, wheel.Metadata{
		Name:    cmp.Or(params.Name, metaFile.Name),
		Version: cmp.Or(params.Version, metaFile.Version),
		Summary: cmp.Or(params.Description, metaFile.Description),
	})
	if err != nil {
		var missing *wheel.MissingMetadataError
		if errors.As(err, &missing) {
			return nil, nil, missingMetadataError(missing, params.Build)
		}
		return nil, nil, locale.WrapInputError(err, "err_publish_build_metadata", "Could not read the ingredient metadata: {{.V0}}", errs.JoinMessage(err))
	}

	// Fetch and validate the org key before building anything: a private publish
	// is encrypted-required, so fail closed before any byte could be uploaded.
	provider := orgkey.New(r.cfg, r.project.Owner())
	if !provider.Configured() {
		return nil, nil, locale.NewInputError("err_publish_orgkey_unconfigured", "No organization key service is configured, so this private ingredient cannot be encrypted.")
	}
	defer provider.Close()
	key, keyID, err := provider.Key(context.Background())
	if err != nil {
		return nil, nil, locale.WrapInputError(err, "err_publish_orgkey_unavailable", "Could not obtain the organization key, so nothing was uploaded: {{.V0}}", errs.JoinMessage(err))
	}

	archivePath, cleanup, err := buildWrappedArtifact(params.Build, *meta, key, keyID)
	if err != nil {
		return nil, nil, errs.Wrap(err, "Could not build encrypted artifact")
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
	// The platform rejects description changes on existing ingredients.
	if params.Description == "" && !params.Edit {
		params.Description = meta.Summary
	}
	return meta, cleanup, nil
}

// missingMetadataError tells the user which of name and version are missing, why pyproject.toml did not supply them,
// and where to set them instead.
func missingMetadataError(missing *wheel.MissingMetadataError, srcDir string) error {
	var lines []string
	for _, field := range missing.Fields {
		var reason string
		switch {
		case missing.Pyproject == "":
			reason = locale.Tl("publish_metadata_no_pyproject", "there is no pyproject.toml in {{.V0}}", srcDir)
		case slices.Contains(missing.Dynamic, field):
			reason = locale.Tl("publish_metadata_dynamic",
				"{{.V0}} lists it in [project].dynamic, so it is computed by a build backend and cannot be read from the file",
				missing.Pyproject)
		default:
			reason = locale.Tl("publish_metadata_absent", "{{.V0}} has no '{{.V1}}' in its [project] table", missing.Pyproject, field)
		}
		lines = append(lines, locale.Tl("publish_metadata_missing_field",
			" - [ACTIONABLE]{{.V0}}[/RESET]: {{.V1}}. Set it in pyproject.toml, with '[ACTIONABLE]--{{.V0}}[/RESET]', or as '[ACTIONABLE]{{.V0}}:[/RESET]' in a '[ACTIONABLE]--meta[/RESET]' file.",
			field, reason))
	}
	return locale.NewInputError("err_publish_build_metadata_missing",
		"Could not determine the ingredient {{.V0}}:\n{{.V1}}", strings.Join(missing.Fields, " and "), strings.Join(lines, "\n"))
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

const (
	privateBuilderNamespace = "builder"
	privateBuilderName      = "private-builder"
)

// addPrivateBuilderDependency makes a --build publish depend on the
// private-builder. Without a builder dependency the platform falls back to the
// noop-builder, which produces an empty artifact. An --edit revision already
// inherits it from the previous revision, so it is only added when missing.
func addPrivateBuilderDependency(reqVars *request.PublishVariables) {
	for _, dep := range reqVars.Dependencies {
		if dep.Namespace == privateBuilderNamespace && dep.Name == privateBuilderName {
			return
		}
	}
	reqVars.Dependencies = append(reqVars.Dependencies, request.PublishVariableDep{
		Dependency: request.Dependency{
			Name:                privateBuilderName,
			Namespace:           privateBuilderNamespace,
			VersionRequirements: ">=0",
		},
	})
}

// buildWrappedArtifact packs srcDir into a wheel under the given metadata,
// encrypts it under the org key, and wraps the ciphertext in a tar.gz ready for
// upload. It returns the wrapped archive path and a cleanup function the caller
// must invoke once the upload is done.
//
// Only the ciphertext ever reaches the wrapped archive: the plaintext wheel and
// payload are removed before the function returns, so no plaintext outlives the
// build.
func buildWrappedArtifact(srcDir string, meta wheel.Metadata, key []byte, keyID string) (archivePath string, cleanup func(), rerr error) {
	tmpDir, err := os.MkdirTemp("", "state-publish-build-")
	if err != nil {
		return "", nil, errs.Wrap(err, "Could not create temp dir")
	}
	removeTmpDir := func() { _ = os.RemoveAll(tmpDir) }
	defer func() {
		if rerr != nil {
			removeTmpDir()
		}
	}()
	cleanup = removeTmpDir

	wheelPath, err := wheel.Pack(srcDir, meta, tmpDir)
	if err != nil {
		return "", nil, errs.Wrap(err, "Could not build a wheel from %s", srcDir)
	}

	// Assemble the tar.gz that becomes the encrypted payload, with the wheel at
	// its root.
	plaintextPayload := filepath.Join(tmpDir, "payload.tar.gz")
	if err := archiver.CreateTgz(plaintextPayload, tmpDir, []archiver.FileMap{
		{Source: wheelPath, Target: filepath.Base(wheelPath)},
	}); err != nil {
		return "", nil, errs.Wrap(err, "Could not assemble payload")
	}

	ciphertextPath := filepath.Join(tmpDir, artifactcrypto.PayloadFilename)
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

	archivePath = filepath.Join(tmpDir, "ingredient.tar.gz")
	if err := archiver.CreateTgz(archivePath, tmpDir, []archiver.FileMap{
		{Source: ciphertextPath, Target: artifactcrypto.PayloadFilename},
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
