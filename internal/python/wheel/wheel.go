// Package wheel builds spec-compliant, byte-reproducible pure-Python wheels from
// a local source tree. It only relocates and zips files; it never executes the
// source it packs (no setup.py, no PEP 517 backend, no subprocess of any kind).
package wheel

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

var (
	// ErrNoPythonFiles indicates the source tree has no .py files to pack.
	ErrNoPythonFiles = errs.New("source tree contains no Python files")
	// ErrNativeContent indicates the source tree contains compiled, platform-specific files.
	ErrNativeContent = errs.New("source tree contains non-pure-Python (compiled) files")
	// ErrMissingMetadata indicates the package name or version could not be determined.
	ErrMissingMetadata = errs.New("package name and version are required")
)

// fixedModTime is stamped on every zip entry so output never depends on on-disk
// timestamps. It is the earliest time the zip format can represent.
var fixedModTime = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

// Metadata is the core package metadata recorded in the wheel. Empty fields fall
// back to pyproject.toml; non-empty fields override it.
type Metadata struct {
	Name    string
	Version string
	Summary string
}

// sourceFile is one file destined for the wheel, identified by its slash path
// relative to both the source tree and the wheel root.
type sourceFile struct {
	rel string
	abs string
}

// Pack builds a pure-Python wheel from srcDir, writes it into outDir as
// {normalized_name}-{version}-py3-none-any.whl, and returns the wheel path.
//
// The wheel root mirrors srcDir: the caller points srcDir at the directory whose
// children are the importable packages. The top-level pyproject.toml (read for
// metadata), __pycache__ directories, *.pyc/*.pyo files, and version-control
// directories are not packed. Compiled files (.so/.pyd/.dylib) are rejected.
// Values in meta override those read from pyproject.toml; the name and version
// must resolve from one source or the other.
//
// Output is byte-reproducible: identical input trees produce identical wheels
// regardless of file timestamps. On any failure no wheel is left at the path.
func Pack(srcDir string, meta Metadata, outDir string) (_ string, rerr error) {
	resolved, err := resolveMetadata(srcDir, meta)
	if err != nil {
		return "", errs.Wrap(err, "could not resolve package metadata")
	}

	files, err := collectFiles(srcDir)
	if err != nil {
		return "", errs.Wrap(err, "could not scan source tree")
	}

	outPath := filepath.Join(outDir, wheelFilename(resolved.Name, resolved.Version))
	if err := writeWheel(files, resolved, outPath); err != nil {
		return "", errs.Wrap(err, "could not write wheel")
	}
	return outPath, nil
}

// collectFiles walks srcDir and returns the files to pack, sorted by their wheel
// path. Cruft is skipped, compiled content is rejected, and at least one .py file
// must be present.
func collectFiles(srcDir string) ([]sourceFile, error) {
	var files []sourceFile
	hasPython := false

	err := filepath.WalkDir(srcDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != srcDir && isExcludedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(srcDir, p)
		if err != nil {
			return errs.Wrap(err, "could not relativize path")
		}
		rel = filepath.ToSlash(rel)

		if isExcludedFile(rel) {
			return nil
		}
		if isNativeFile(rel) {
			return errs.Wrap(ErrNativeContent, "offending file: %s", rel)
		}
		if strings.ToLower(path.Ext(rel)) == ".py" {
			hasPython = true
		}
		files = append(files, sourceFile{rel: rel, abs: p})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !hasPython {
		return nil, ErrNoPythonFiles
	}
	sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })
	return files, nil
}

func isExcludedDir(name string) bool {
	switch name {
	case "__pycache__", ".git", ".hg", ".svn":
		return true
	}
	return false
}

func isExcludedFile(rel string) bool {
	if rel == "pyproject.toml" { // top-level only: metadata source, not package data
		return true
	}
	switch strings.ToLower(path.Ext(rel)) {
	case ".pyc", ".pyo":
		return true
	}
	return false
}

// isNativeFile reports whether rel is a compiled, platform-specific shared
// library that has no place in a pure py3-none-any wheel. Extensions are matched
// case-insensitively because Windows filesystems are.
func isNativeFile(rel string) bool {
	switch strings.ToLower(path.Ext(rel)) {
	case ".so", ".pyd", ".dll", ".dylib":
		return true
	}
	return false
}

// writeWheel writes the wheel to a sibling temp file and renames it onto outPath
// only after the whole archive is written, so a failure leaves outPath untouched.
func writeWheel(files []sourceFile, meta resolvedMetadata, outPath string) (rerr error) {
	tmp, err := os.CreateTemp(filepath.Dir(outPath), filepath.Base(outPath)+".tmp-*")
	if err != nil {
		return errs.Wrap(err, "could not create temp wheel")
	}
	tmpName := tmp.Name()
	defer func() {
		if rerr == nil {
			return
		}
		if tmp != nil {
			if err := tmp.Close(); err != nil {
				rerr = errs.Pack(rerr, errs.Wrap(err, "could not close temp wheel"))
			}
		}
		if err := os.Remove(tmpName); err != nil {
			rerr = errs.Pack(rerr, errs.Wrap(err, "could not remove temp wheel"))
		}
	}()

	wheelHash := sha256.New()
	zw := zip.NewWriter(io.MultiWriter(tmp, wheelHash))

	distInfo := distInfoDir(meta.Name, meta.Version)

	// Source files plus METADATA and WHEEL form the recorded entry set. RECORD is
	// written last because it lists the hashes of all the others.
	entries := make([]wheelEntry, 0, len(files)+2)
	for _, f := range files {
		entries = append(entries, wheelEntry{name: f.rel, abs: f.abs})
	}
	entries = append(entries,
		wheelEntry{name: distInfo + "/METADATA", data: buildMetadata(meta)},
		wheelEntry{name: distInfo + "/WHEEL", data: buildWheelFile()},
	)
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	records := make([]record, 0, len(entries)+1)
	for _, e := range entries {
		hash, size, err := e.write(zw)
		if err != nil {
			return errs.Wrap(err, "could not add entry %s", e.name)
		}
		records = append(records, record{name: e.name, hash: hash, size: size})
	}

	recordName := distInfo + "/RECORD"
	if _, _, err := (wheelEntry{name: recordName, data: buildRecord(records, recordName)}).write(zw); err != nil {
		return errs.Wrap(err, "could not add RECORD")
	}

	if err := zw.Close(); err != nil {
		return errs.Wrap(err, "could not finalize zip")
	}
	if err := tmp.Close(); err != nil {
		tmp = nil
		return errs.Wrap(err, "could not close temp wheel")
	}
	tmp = nil
	if err := os.Rename(tmpName, outPath); err != nil {
		return errs.Wrap(err, "could not finalize wheel")
	}

	logging.Debug("Packed wheel %s: %d files, sha256=%s", filepath.Base(outPath), len(files), hex.EncodeToString(wheelHash.Sum(nil)))
	return nil
}

// wheelEntry is a single archive member, sourced either from disk (abs set) or
// from memory (data set).
type wheelEntry struct {
	name string
	abs  string
	data []byte
}

// write adds the entry to zw with fixed metadata and returns its PEP 376 hash
// string and size.
func (e wheelEntry) write(zw *zip.Writer) (hash string, size int64, rerr error) {
	fh := &zip.FileHeader{Name: e.name, Method: zip.Deflate, Modified: fixedModTime}
	fh.SetMode(0644)
	w, err := zw.CreateHeader(fh)
	if err != nil {
		return "", 0, errs.Wrap(err, "could not create zip entry")
	}

	var r io.Reader
	if e.abs != "" {
		f, err := os.Open(e.abs)
		if err != nil {
			return "", 0, errs.Wrap(err, "could not open source file")
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				rerr = errs.Pack(rerr, errs.Wrap(cerr, "could not close source file"))
			}
		}()
		r = f
	} else {
		r = bytes.NewReader(e.data)
	}

	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(w, h), r)
	if err != nil {
		return "", 0, errs.Wrap(err, "could not write zip entry")
	}
	return "sha256=" + base64.RawURLEncoding.EncodeToString(h.Sum(nil)), n, nil
}
