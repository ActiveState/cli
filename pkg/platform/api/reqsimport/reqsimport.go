package reqsimport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
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
func Init() *ReqsImport {
	svcURL := api.GetServiceURL(api.ServiceRequirementsImport)
	url := svcURL.Scheme + path.Join(svcURL.Host, svcURL.Path)

	opts := Opts{
		TranslateURL: url,
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

	logging.Debug("POSTing data to reqsvc")
	resp, err := ri.client.Post(url, translateContentType, &buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint

	var respMsg ReqsTxtTranslateRespMsg
	if err := json.NewDecoder(resp.Body).Decode(&respMsg); err != nil {
		return nil, err
	}

	if len(respMsg.LineErrs) > 0 {
		return nil, &TranslateResponseError{respMsg.LineErrs}
	}

	return respMsg.Changeset, nil
}

// ReqsTxtTranslateReqMsg ...
type ReqsTxtTranslateReqMsg struct {
	Data string `json:"requirements"`
}

// ReqsTxtTranslateRespMsg ...
type ReqsTxtTranslateRespMsg struct {
	Changeset model.Changeset      `json:"changeset,omitempty"`
	LineErrs  []TranslateLineError `json:"errors,omitempty"`
}

// TranslateResponseError ...
type TranslateResponseError struct {
	LineErrs []TranslateLineError
}

func (e *TranslateResponseError) Error() string {
	var msgs, sep string
	for _, lineErr := range e.LineErrs {
		msgs += sep + lineErr.Error()
		sep = "; "
	}
	if msgs == "" {
		msgs = "unknown error"
	}

	return locale.Tr("reqsvc_err_line_errors", msgs)
}

// TranslateLineError ...
type TranslateLineError struct {
	ErrMsg string `json:"errorText,omitempty"`
	PkgTxt string `json:"packageText,omitempty"`
}

func (e *TranslateLineError) Error() string {
	return fmt.Sprintf("line %q: %s", e.PkgTxt, e.ErrMsg)
}
