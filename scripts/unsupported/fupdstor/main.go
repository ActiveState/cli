package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
)

func main() {
	if err := run(); err != nil {
		cmd := path.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "%s: %v\n", cmd, err)
		os.Exit(1)
	}
}

//go:embed assets
var assets embed.FS

func run() error {
	var (
		port    = ":8686"
		verbose bool
	)

	flag.StringVar(&port, "p", port, "port to serve from")
	flag.BoolVar(&verbose, "v", verbose, "log requests")
	flag.Parse()

	fs := http.FS(assets)
	fs = &prefixedFS{
		prefix: "/assets",
		fs:     fs,
	}

	var m http.Handler
	m = http.FileServer(fs)

	if verbose {
		m = logRequests(m)
	}

	return http.ListenAndServe(port, m)
}

type prefixedFS struct {
	prefix string
	fs     http.FileSystem
}

func (fs *prefixedFS) Open(name string) (http.File, error) {
	name = fs.prefix + name
	return fs.fs.Open(name)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s ", r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}
