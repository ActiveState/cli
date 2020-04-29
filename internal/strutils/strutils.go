package strutils

import (
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
)

func UUID() strfmt.UUID {
	return strfmt.UUID(uuid.New().String())
}
