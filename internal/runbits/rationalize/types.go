package rationalize

import "errors"

// Inner is just an alias because otherwise external use of this struct would not be able to construct the error
// property, and we want to keep the boilerplate minimal.
type Inner error

// ErrNoProject communicates that we were unable to find an activestate.yaml
var ErrNoProject = errors.New("no project")

// ErrNotAuthenticated communicates that the user is not logged in
var ErrNotAuthenticated = errors.New("not authenticated")

var ErrActionAborted = errors.New("aborted by user")
