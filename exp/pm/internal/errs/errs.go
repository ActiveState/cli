package errs

type DoneError interface {
	error
	DoneMsg() string
}
