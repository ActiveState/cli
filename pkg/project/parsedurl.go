package project

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// ParsedURL represents a project url
type ParsedURL struct {
	Owner     string
	Project   string
	Namespace *Namespace
	CommitID  *strfmt.UUID
	Branch    string
	url       *url.URL
}

type ConfigAble interface {
	projectfile.ConfigGetter
}

// Set implements the captain argmarshaler interface.
func (u *ParsedURL) Set(v string) error {
	if u == nil {
		return fmt.Errorf("cannot set nil value")
	}

	parsedNs, err := NewParsedURL(v)
	if err != nil {
		return err
	}

	*u = *parsedNs
	return nil
}

// String implements the fmt.Stringer interface.
func (u *ParsedURL) String() string {
	if u == nil {
		return ""
	}

	return u.url.String()
}

// Type returns the human readable type name of ParsedURL.
func (u *ParsedURL) Type() string {
	return "namespace"
}

// IsValid returns whether or not the namespace is set sufficiently.
func (u *ParsedURL) IsValid() bool {
	return u != nil && u.Owner != "" && u.Project != ""
}

// Validate returns a failure if the namespace is not valid.
func (u *ParsedURL) Validate() error {
	if u == nil || !u.IsValid() {
		return locale.NewInputError("err_invalid_namespace", "", u.String())
	}
	return nil
}

// NewParsedURL returns a valid project namespace
func NewParsedURL(raw string) (*ParsedURL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, &ErrorParseProject{locale.NewError("err_bad_project_url")}
	}

	parsedUrl := &ParsedURL{url: u}

	pathBits := strings.Split("/", u.Path)
	if len(pathBits) == 2 {
		parsedUrl.Owner = pathBits[0]
		parsedUrl.Project = pathBits[1]
		parsedUrl.Namespace = NewNamespace(parsedUrl.Owner, parsedUrl.Project)
	}

	query := u.Query()

	if v, ok := query["commitID"]; ok && len(v) > 0 {
		if len(v) > 1 {
			return nil, &ErrorParseProject{locale.NewError("err_bad_project_url")}
		}
		vv := strfmt.UUID(v[0])
		parsedUrl.CommitID = &vv
	}

	if v, ok := query["branch"]; ok && len(v) > 0 {
		if len(v) > 1 {
			return nil, &ErrorParseProject{locale.NewError("err_bad_project_url")}
		}
		parsedUrl.Branch = v[0]
	}

	if parsedUrl.CommitID != nil {
		if err := validateUUID((*parsedUrl.CommitID).String()); err != nil {
			return parsedUrl, err
		}
	}

	return parsedUrl, nil
}
