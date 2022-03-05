package serve

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
)

type Serve struct {
	h    *http.Server
	errs chan error
}

func New() *Serve {
	return &Serve{
		h: &http.Server{
			Handler: NewEndpoints(),
		},
	}
}

func (s *Serve) Run() (string, error) {
	emsg := "serve run: %w"

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf(emsg, err)
	}

	if s.errs == nil {
		s.errs = make(chan error)
	}

	go func() {
		defer close(s.errs)
		s.errs <- s.h.Serve(l)
	}()

	return l.Addr().String(), nil
}

func (s *Serve) Wait() error {
	if err := <-s.errs; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Serve) Close() error {
	return s.h.Shutdown(context.Background())
}

var handleInfoPath = "/info"

type Endpoints struct {
	h http.Handler
}

func NewEndpoints() *Endpoints {
	m := http.NewServeMux()
	m.HandleFunc(handleInfoPath, handleInfo)

	return &Endpoints{
		h: m,
	}
}

func (es *Endpoints) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	es.h.ServeHTTP(w, r)
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "DATA")
}

type Client struct {
	addr string
	c    *http.Client
}

func NewClient(addr string) *Client {
	return &Client{
		addr: addr,
		c:    &http.Client{},
	}
}

func (c *Client) GetInfo() (string, error) {
	r, err := c.c.Get("http://" + c.addr + handleInfoPath)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
