package gql

import (
	"errors"
	"time"

	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/go-openapi/strfmt"
)

var (
	ErrNoValueAvailable       = errors.New("no value available")
	ErrMissingBranchProjectID = errors.New("missing branch proj id")
)

func makeStrfmtDateTime(t time.Time) strfmt.DateTime {
	dt, err := strfmt.ParseDateTime(t.Format(time.RFC3339))
	if err != nil {
		panic(err) // this should never happen
	}
	return dt
}

func newStrfmtURI(s *string) *strfmt.URI {
	if s == nil || *s == "" {
		return nil
	}

	var uri strfmt.URI
	if err := uri.UnmarshalText([]byte(*s)); err != nil {
		panic(err) // should only happen if the backend is storing bad data
	}
	return &uri
}

func forkedProjectToMonoForkedFrom(fp *ForkedProject) *mono_models.ProjectForkedFrom {
	if fp == nil {
		return nil
	}
	return &mono_models.ProjectForkedFrom{
		Project:      fp.Name,
		Organization: fp.Organization.URLName,
	}
}

func ptrStrfmtUUIDToPtrString(id *strfmt.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

func ptrBoolToBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
