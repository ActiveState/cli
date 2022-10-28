package installmgr

import (
	"errors"

	"golang.org/x/sys/windows"
)

func isAccessDeniedError(err error) bool {
	return errors.Is(err, windows.ERROR_ACCESS_DENIED)
}
