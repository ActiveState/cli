package rationalize

// Inner is just an alias because otherwise external use of this struct would not be able to construct the error
// property, and we want to keep the boilerplate minimal.
type Inner error

type ErrNoProject struct {
	Inner
}
