package captain

type ErrorWithExitCode interface {
	Error() string
	ExitCode() int
}

func NewError(message string, exitCode int) Error {
	return Error{message, exitCode}
}

type Error struct {
	message  string
	exitCode int
}

func (e Error) Error() string {
	return e.message
}

func (e Error) ExitCode() int {
	return e.exitCode
}
