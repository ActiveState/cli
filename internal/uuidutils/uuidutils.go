package uuidutils

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/go-openapi/strfmt"
)

func ValidateUUID(uuidStr string) (strfmt.UUID, error) {
	var uuid strfmt.UUID
	if ok := strfmt.Default.Validates("uuid", uuidStr); !ok {
		return uuid, locale.NewError("invalid_uuid_val", "Invalid UUID {{.V0}} value.", uuidStr)
	}

	if err := uuid.UnmarshalText([]byte(uuidStr)); err != nil {
		return uuid, locale.WrapError(err, "err_uuid_unmarshal", "Failed to unmarshal the uuid {{.V0}}.", uuidStr)
	}

	return uuid, nil
}
