package reqsimport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path"
	"time"

	"github.com/ActiveState/cli/pkg/platform/model"
)

const (
	translateContentType = "application/json"
)

// Opts ...
type Opts struct {
	TranslateURL string
}

// ReqsImport ...
type ReqsImport struct {
	opts   Opts
	client *http.Client
}

// New ...
func New(opts Opts) (*ReqsImport, error) {
	c := &http.Client{
		Timeout: 60 * time.Second,
	}

	ri := ReqsImport{
		opts:   opts,
		client: c,
	}

	return &ri, nil
}

// Init ...
func Init(host, urlPath string) *ReqsImport {
	opts := Opts{
		TranslateURL: "https://" + path.Join(host, urlPath),
	}

	ri, err := New(opts)
	if err != nil {
		panic(err)
	}

	return ri
}

// Changeset ...
func (ri *ReqsImport) Changeset(data []byte) (model.Changeset, error) {
	reqMsg := ReqsTxtTranslateReqMsg{
		Data: string(data),
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(reqMsg); err != nil {
		return nil, err
	}

	url := ri.opts.TranslateURL

	resp, err := ri.client.Post(url, translateContentType, &buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint

	var respMsg ReqsTxtTranslateRespMsg

	if err := json.NewDecoder(resp.Body).Decode(&respMsg); err != nil {
		return nil, err
	}

	return respMsg.CommitRequest.Changeset, nil
}

// ReqsTxtTranslateReqMsg ...
type ReqsTxtTranslateReqMsg struct {
	Data string `json:"requirements"`
}

// ReqsTxtTranslateRespMsg ...
type ReqsTxtTranslateRespMsg struct {
	*model.CommitRequest
	Errors []string `json:"errors,omitempty"`
}
