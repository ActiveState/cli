package srvstatus

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"
)

type SrvStatus struct {
	*httptest.Server
	h *Handle
}

func New() *SrvStatus {
	h := &Handle{}

	return &SrvStatus{
		Server: httptest.NewServer(h),
		h:      h,
	}
}

type Handle struct {
}

func (h *Handle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	segs := strings.SplitN(path, "/", 2)
	codeSeg := segs[0]
	code, err := strconv.Atoi(codeSeg)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "first segment malformed: should be valid http status code")
		return
	}

	sleepParam := r.URL.Query().Get("sleep")
	if sleepParam != "" {
		sleepLen, err := strconv.Atoi(sleepParam)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "sleep param malformed: should be integer")
			return
		}

		time.Sleep(time.Millisecond * time.Duration(sleepLen))
	}

	w.WriteHeader(code)
}
