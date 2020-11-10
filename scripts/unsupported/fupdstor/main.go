// fupdstor provides a fake update file server based on the curent expectations
// of the state tool.
//
// Example usage (set bin dir and log requests):
//   fupdstor -d ../../../build -v
//
// This requires updating the const APIUpdateURL value to something like
// "http://localhost:8686/cli-update/update/"
package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	if err := run(); err != nil {
		cmd := path.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "%s: %v\n", cmd, err)
		os.Exit(1)
	}
}

func run() error {
	var (
		file    = "state"
		dir     string
		port    = ":8686"
		verbose bool
	)

	flag.StringVar(&dir, "d", dir, "directory to find bin")
	flag.StringVar(&port, "p", port, "port to serve from")
	flag.BoolVar(&verbose, "v", verbose, "log requests")
	flag.Parse()

	if _, err := os.Stat(dir); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		file += ".exe"
	}

	filePath := filepath.Join(dir, file)

	var m http.Handler
	m = &extMux{
		json: &versionInfo{
			file: filePath,
		},
		comp: &fileComp{
			file: filePath,
		},
	}

	if verbose {
		m = logRequests(m)
	}

	return http.ListenAndServe(port, m)
}

var errNotFound = errors.New("not found")

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s ", r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

type extMux struct {
	json http.Handler
	comp http.Handler
}

func (m *extMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ext := path.Ext(r.URL.Path)

	switch ext {
	case ".json":
		m.json.ServeHTTP(w, r)
	case ".gz", ".zip":
		m.comp.ServeHTTP(w, r)
	default:
		writeSimple(w, http.StatusNotFound)
	}
}

type versionInfo struct {
	file string
}

// https://s3.ca-central-1.amazonaws.com/cli-update/update/state/dgreen/fix_autoupd_trig_activation_err-175607307/linux-amd64.json
func (v *versionInfo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open(v.file)
	if err != nil {
		logIntServErr(w, "version info", err)
		return
	}

	var buf bytes.Buffer

	ext := ".gz"
	if strings.HasPrefix(path.Base(r.URL.Path), "windows") {
		ext = ".zip"
	}

	if err := applyComp(&buf, ext, f); err != nil {
		if errors.Is(err, errNotFound) {
			writeSimple(w, http.StatusNotFound)
			return
		}

		logIntServErr(w, "file comp", err)
		return
	}

	sha, err := generateSha256(&buf)
	if err != nil {
		logIntServErr(w, "version info", err)
	}

	info := updateInfo{
		Version:  "9000.0.1-123",
		Sha256v2: sha,
	}

	if err := json.NewEncoder(w).Encode(&info); err != nil {
		log.Printf("handle json: %v\n", err)
	}
}

// make type to store file location flag data
type fileComp struct {
	file string
}

// https://s3.ca-central-1.amazonaws.com/cli-update/update/state/dgreen/fix_autoupd_trig_activation_err-175607307/9000.0.1-123/linux-amd64.gz
func (c *fileComp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open(c.file)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("file comp: file %q does not exist\n", c.file)
			writeSimple(w, http.StatusNotFound)
			return
		}

		logIntServErr(w, "file comp", err)
		return
	}

	var buf bytes.Buffer

	if err := applyComp(&buf, path.Ext(r.URL.Path), f); err != nil {
		if errors.Is(err, errNotFound) {
			writeSimple(w, http.StatusNotFound)
			return
		}

		logIntServErr(w, "file comp", err)
		return
	}

	if _, err = io.Copy(w, &buf); err != nil {
		log.Printf("file comp: %v\n", err)
		return
	}
}

func applyComp(buf io.Writer, ext string, f *os.File) error {
	var wc writeClose

	switch ext {
	case ".gz":
		info, err := f.Stat()
		if err != nil {
			return fmt.Errorf("compress file: %w", err)
		}

		g := gzip.NewWriter(buf)
		t := tar.NewWriter(g)
		hdr := tar.Header{
			Name: info.Name(),
			Mode: int64(info.Mode()),
			Size: info.Size(),
		}
		if err := t.WriteHeader(&hdr); err != nil {
			return fmt.Errorf("compress file: %w", err)
		}
		wc.Writer = t
		wc.closers = []io.Closer{t, g}

	case ".zip":
		var err error
		z := zip.NewWriter(buf)
		wc.Writer, err = z.Create(filepath.Base(f.Name()))
		if err != nil {
			return fmt.Errorf("compress file: %w", err)
		}
		wc.closers = []io.Closer{z}

	default:
		return fmt.Errorf("compress file: %w", errNotFound)
	}

	if _, err := io.Copy(wc, f); err != nil {
		return fmt.Errorf("compress file: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("compress file: %w", err)
	}

	return nil
}

func writeSimple(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
	fmt.Fprintln(w, http.StatusText(code))
}

func logIntServErr(w http.ResponseWriter, scope string, err error) {
	log.Printf("%s: %v\n", scope, err)
	writeSimple(w, http.StatusInternalServerError)
}

func generateSha256(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("generate sha256: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

type writeClose struct {
	io.Writer
	closers []io.Closer
}

func (wc *writeClose) Close() error {
	for _, c := range wc.closers {
		if err := c.Close(); err != nil {
			return fmt.Errorf("writeclose close: %w", err)
		}
	}
	return nil
}

/*{
    "Version": "0.2.2-957",
    "Sha256v2": "832b51..."
}*/
type updateInfo struct {
	Version  string
	Sha256v2 string
}
