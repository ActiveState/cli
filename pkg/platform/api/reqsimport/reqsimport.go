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
	jsonContentType = "application/json"
)

// Opts contains the options available for the primary package type ReqsImport.
type Opts struct {
	ReqsvcURL string
}

// ReqsImport represents a reusable http.Client and related options.
type ReqsImport struct {
	opts   Opts
	client *http.Client
}

// New forms a pointer to a default ReqsImport instance.
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

// Init is a convenience wrapper for New.
func Init() *ReqsImport {
	svcURL := api.GetServiceURL(api.ServiceRequirementsImport)
	url := svcURL.Scheme + "://" + path.Join(svcURL.Host, svcURL.Path)

	opts := Opts{
		ReqsvcURL: url,
	}

	ri, err := New(opts)
	if err != nil {
		panic(err)
	}

	return ri
}

// Changeset posts requirements data to a backend service and returns a
// Changeset that can be committed to a project.
func (ri *ReqsImport) Changeset(data []byte, lang string) (model.Changeset, error) {
	reqPayload := &TranslationReqMsg{
		Data:     string(data),
		Language: lang,
	}
	respPayload := &TranslationRespMsg{}

	err := postJSON(ri.client, ri.opts.ReqsvcURL, reqPayload, respPayload)
	if err != nil {
		return nil, err
	}

	if len(respPayload.LineErrs) > 0 {
		return nil, &TranslationResponseError{respPayload.LineErrs}
	}

	return respPayload.Changeset, nil
}

// TranslationReqMsg represents the message sent to the requirements
// translation service.
type TranslationReqMsg struct {
	Data     string `json:"requirements"`
	Language string `json:"language"`
}

// TranslationRespMsg represents the message returned by the requirements
// translation service.
type TranslationRespMsg struct {
	Changeset model.Changeset        `json:"changeset,omitempty"`
	LineErrs  []TranslationLineError `json:"errors,omitempty"`
}

// TranslationLineError represents an error reported by the requirements
// translation service regarding a single line processed from the request.
type TranslationLineError struct {
	ErrMsg string `json:"errorText,omitempty"`
	PkgTxt string `json:"packageText,omitempty"`
}

// Error implements the error interface.
func (e *TranslationLineError) Error() string {
	return fmt.Sprintf("line %q: %s", e.PkgTxt, e.ErrMsg)
}

// TranslationResponseError contains multiple error messages and allows them to
// be handled as a common error.
type TranslationResponseError struct {
	LineErrs []TranslationLineError
}

// Error implements the error interface.
func (e *TranslationResponseError) Error() string {
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

func postJSON(client *http.Client, url string, reqPayload, respPayload interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(reqPayload); err != nil {
		return err
	}

	logging.Debug("POSTing JSON")
	resp, err := client.Post(url, jsonContentType, &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint

	return json.NewDecoder(resp.Body).Decode(&respPayload)
}
