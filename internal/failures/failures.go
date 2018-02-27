package failures

import (
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
)

// Failure defines the main interface used by all our custom failure structs
type Failure interface {
	Error() string
	Handle(string)
}

type app struct{}

// New creates a new failure struct
func (e *app) New(msg string) Failure {
	return &AppFailure{msg}
}

// App failure is used to easily create an app facing failure (failures.App.New(...))
var App = app{}

// AppFailure is the actual struct used for the failure, so what failures.App.New() creates
type AppFailure struct {
	message string
}

// Error returns the failure message
func (e *AppFailure) Error() string {
	return e.message
}

// Handle handles the error message, this is used to communicate that the error occurred in whatever fashion is
// most relevant to the current error type
func (e *AppFailure) Handle(description string) {
	if description == "" {
		description = "App Error:"
	} else {
		print.Error(description)
	}
	logging.Error(description)
	logging.Error(e.Error())
}

type user struct{}

// New creates a new failure struct
func (e *user) New(msg string) Failure {
	return &UserFailure{msg}
}

// User failure is used to easily create an user facing failure (failures.User.New(...))
var User = user{}

// UserFailure is the actual struct used for the failure, so what failures.User.New() creates
type UserFailure struct {
	message string
}

// Error returns the failure message
func (e *UserFailure) Error() string {
	return e.message
}

// Handle handles the error message, this is used to communicate that the error occurred in whatever fashion is
// most relevant to the current error type
func (e *UserFailure) Handle(description string) {
	if description != "" {
		logging.Error(description)
		print.Error(description)
	}
	logging.Error(e.Error())
	print.Error(e.Error())
}

// Handle is what controllers would call to handle an error message, this will take care of calling the underlying
// handle method or logging the error if this isnt a Failure type
func Handle(err error, description string) {
	switch t := err.(type) {
	default:
		if description == "" {
			description = "Unknown Error:"
		} else {
			print.Error(description)
		}
		logging.Error(description)
		logging.Error(err.Error())
		return
	case Failure:
		t.Handle(description)
		return
	}
}
