package rationalize

// Inner is just an alias because otherwise external use of this struct would not be able to construct the error
// property, and we want to keep the boilerplate minimal.
type Inner error

type ErrNoProject struct {
	Inner
}

type ErrNotAuthenticated struct {
	Inner
}

type ErrActionAborted struct {
	Inner
}

type ErrPermission struct {
	Inner
	Details interface{}
}
