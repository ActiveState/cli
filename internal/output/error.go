package output

// Error describes the behavior of JSONRPC-based error types.
type Error interface {
	error
	ReturnCode() int
	UserError() string
	ErrorData() string
}

// GeneralError is a wrapper type used to transform error implementations into
// a structure useful for serializable output.
type GeneralError struct {
	Err     error  `json:"-"`
	RPCCode int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Data    string `json:"data,omitempty"`
}

// NewGeneralError transforms an error into a GeneralError instance. If any
// Error methods are not available, a simple default is assigned.
func NewGeneralError(err error) *GeneralError {
	if err == nil {
		return nil
	}

	gerr := GeneralError{
		Err:     err,
		RPCCode: 1,
		Message: err.Error(),
	}

	if coder, ok := err.(interface{ ReturnCode() int }); ok {
		gerr.RPCCode = coder.ReturnCode()
	}

	if uerr, ok := err.(interface{ UserError() string }); ok {
		gerr.Message = uerr.UserError()
	}

	if edProvider, ok := err.(interface{ ErrorData() string }); ok {
		gerr.Data = edProvider.ErrorData()
	}

	return &gerr
}

// Error implements the error interface.
func (e *GeneralError) Error() string {
	msg := e.Err.Error()
	if e.Data != "" {
		msg += " (" + e.Data + ")"
	}
	return msg
}

// ReturnCode returns the RPC return code value.
func (e *GeneralError) ReturnCode() int {
	return e.RPCCode
}

// UserError returns basic and sanitized error information suitable for users.
func (e *GeneralError) UserError() string {
	return e.Message
}

// ErrorData returns extra information regarding an error in the form of a
// string.
func (e *GeneralError) ErrorData() string {
	return e.Data
}

// Unwrap facilitates error unwrapping.
func (e *GeneralError) Unwrap() error {
	return e.Err
}
