package pcerrors

type DoneError interface {
	error
	DoneMsg() string
}
