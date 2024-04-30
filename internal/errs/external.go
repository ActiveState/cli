package errs

type ExternalError interface {
	ExternalError() bool
}

func IsExternalError(err error) bool {
	if err == nil {
		return false
	}

	for _, err := range Unpack(err) {
		errExternal, ok := err.(ExternalError)
		if ok && errExternal.ExternalError() {
			return true
		}
	}

	return false
}
