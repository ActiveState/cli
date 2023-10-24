package rationalize

import (
	"errors"
	"fmt"
)

// Inner is just an alias because otherwise external use of this struct would not be able to construct the error
// property, and we want to keep the boilerplate minimal.
type Inner error

// ErrNoProject communicates that we were unable to find an activestate.yaml
var ErrNoProject = errors.New("no project")

// ErrNotAuthenticated communicates that the user is not logged in
var ErrNotAuthenticated = errors.New("not authenticated")

var ErrActionAborted = errors.New("aborted by user")

var ErrHeadless = errors.New("headless")

type ErrAPI struct {
	Wrapped error
	Code    int
	Message string
}

func (e *ErrAPI) Error() string { return fmt.Sprintf("API code %d: %s", e.Code, e.Message) }

func (e *ErrAPI) Unwrap() error { return e.Wrapped }
