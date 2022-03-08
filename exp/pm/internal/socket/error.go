package socket

type DoneError struct {
	doneMsg string
}

func NewDoneError() *DoneError {
	return &DoneError{
		doneMsg: "done",
	}
}

func (e *DoneError) Error() string {
	return e.doneMsg
}

func (e *DoneError) DoneMsg() string {
	return e.doneMsg
}
