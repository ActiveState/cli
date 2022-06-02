// Package relay provides a simple mechanism for relaying control flow based
// upon whether a checked error is nil or not.
package relay

// Relay tracks an error and a related error handler.
type Relay struct {
	err error
	h   func(error)
}

// New constructs a new *Relay.
func New(handler ...func(error)) *Relay {
	h := DefaultHandler()
	if len(handler) > 0 {
		h = handler[0]
	}

	return &Relay{
		h: h,
	}
}

// Check will do nothing if the error argument is nil. Otherwise, it kicks-off
// an event (i.e. the relay is "tripped") and should be handled by a deferred
// call to Handle().
func (r *Relay) Check(err error) {
	if err == nil {
		return
	}

	r.err = err

	panic(r)
}

// CodedCheck will do nothing if the error argument is nil. Otherwise, it
// kicks-off an event (i.e. the relay is "tripped") and should be handled by a
// deferred call to Handle(). Any provided error will be wrapped in a
// codedError instance in order to trigger special behavior in the default
// error handler.
func (r *Relay) CodedCheck(code int, err error) {
	if err == nil {
		return
	}

	r.Check(&CodedError{err, code})
}
