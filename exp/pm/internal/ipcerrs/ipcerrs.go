package ipcerrs

type DoneError interface {
	error
	DoneMsg() string
}
