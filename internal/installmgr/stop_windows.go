package installmgr

import (
	"errors"

	"golang.org/x/sys/windows"
)

func IsAccessDeniedError(err error) bool {
	return errors.Is(err, windows.ERROR_ACCESS_DENIED)
}
