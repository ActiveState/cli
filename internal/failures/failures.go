package failures

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
	"github.com/ActiveState/cli/internal/print"
)

var (
	guid = uuid.New()
	// FailUser identifies a failure as a user facing failure, this doesn't use the Type method as the Type method does
	// not support setting the user bool. This should be the ONLY failure type that does this.
	FailUser = &FailureType{guid.String(), "failures.fail.user", true, []*FailureType{}}

	// failLegacy identifies a failure as a legacy failure, this is for internal use only
	failLegacy = Type("failures.fail.legacy")

	// FailDeveloper identifies a failure as being caused by the developer (eg. a codepath that should never happen unless a developer messed up)
	FailDeveloper = Type("failures.fail.developer")

	// FailIO identifies a failure as an IO failure
	FailIO = Type("failures.fail.io")

	// FailOS identifies a failure as an OS failure
	FailOS = Type("failures.fail.os")

	// FailInput identifies a failure as an input failure
	FailInput = Type("failures.fail.input")

	// FailUserInput identifies a failure as an input failure
	FailUserInput = Type("failures.fail.userinput", FailInput, FailUser)

	// FailCmd identifies a failure as originating from the command that was ran
	FailCmd = Type("failures.fail.cmd", FailUser)

	// FailRuntime identifies a failure as a runtime failure (mainly intended for calls to the runtime package)
	FailRuntime = Type("failures.fail.runtime")

	// FailVerify identifies a failure as being due to the failure to verify something
	FailVerify = Type("failures.fail.verify")

	// FailNotFound identifies a failure as being due to an item not being found
	FailNotFound = Type("failures.fail.notfound")

	// FailNetwork identifies a failure due to a networking issue
	FailNetwork = Type("failures.fail.network")

	// FailTemplating identifies a failure due to a templating issue
	FailTemplating = Type("failures.fail.templating")

	// FailArchiving identifies a failure due to archiving (compressing or decompressing)
	FailArchiving = Type("failures.fail.archiving")

	// FailMarshal identifies a failure due to marshalling or unmarshalling
	FailMarshal = Type("failures.fail.marshal")

	// FailThirdParty identifies a failure due to a third party component (ie. we cannot infer the real reason)
	FailThirdParty = Type("failures.fail.thirdparty")

	// FailInvalidArgument identifies a failure as being due to an argument to a function being invalid.
	FailInvalidArgument = Type("failures.fail.invalid_arg")

	// FailNonFatal is not supposed to be used directly. It communicates a failure that can safely be ignored.
	// Failures that inerhit from this type will not be logged to rollbar.
	FailNonFatal = Type("failures.fail.nonfatal")

	// FailSilent identifies failures that should not produce visible output.
	FailSilent = Type("failures.fail.silent")
)

var handled error

// FailureType reflects a specific type of failure, and is used to identify failures in a generalized way
type FailureType struct {
	UID     string
	Name    string
	User    bool
	parents []*FailureType
}

// Matches tells you if the given FailureType matches the current one or any of its parents
func (f *FailureType) Matches(m *FailureType) bool {
	if f == m {
		return true
	}

	for _, p := range f.parents {
		if p.Matches(m) {
			return true
		}
	}

	return false
}

// New creates a failure struct with the given info
func (f *FailureType) New(message string, params ...string) *Failure {
	var input = map[string]interface{}{}
	for k, v := range params {
		input["V"+strconv.Itoa(k)] = v
	}

	file, line := trace()
	message = locale.T(message, input)

	var logger logging.Logger = logging.Debug
	if !f.Matches(FailUser) && !f.Matches(FailNonFatal) {
		logger = logging.Error
	}
	logger(message)

	return &Failure{message, f, file, line, stacktrace.Get(), nil}
}

// Wrap wraps another error
func (f *FailureType) Wrap(err error, message ...string) *Failure {
	if err == nil {
		return nil
	}

	if len(message) > 0 {
		err = fmt.Errorf("%s: %v", err, strings.Join(message, ": "))
	}
	logging.Debug("Failure '%s' wrapped: %v", f.Name, err)
	fail := f.New(err.Error())
	fail.err = err
	return fail
}

// Failure holds an actual failure, do not call this directly, use Fail and UserFail instead
type Failure struct {
	Message string
	Type    *FailureType
	File    string
	Line    int
	Trace   *stacktrace.Stacktrace
	err     error
}

// Error returns the failure message, cannot be a pointer as it breaks the error interface
func (e Failure) Error() string {
	return e.Message
}

// ToError converts a failure to an error
func (e *Failure) ToError() error {
	if e == nil {
		return nil
	}
	if e.err != nil {
		return e.err
	}
	return e
}

// WithDescription is a convenience method that emulates the behavior of using Handle()
// while allowing the normal propagation of errors up the stack. Instead of sending a
// failure to Handle() and then returning, please add the description with this method
// and use the modified failure as the return value.
func (e *Failure) WithDescription(message string) *Failure {
	e.Message = locale.T(message) + "\n" + e.Message
	return e
}

// Handle handles the error message, this is used to communicate that the error occurred in whatever fashion is
// most relevant to the current error type
//
// If description is empty, only the error message is printed
func (e *Failure) Handle(description string) {
	logging.Debug("Handling failure, Trace:\n %s", e.Trace.String())

	if e.Type.Matches(FailSilent) {
		logging.Debug("Silent failure:\n %s", description)
		return
	}

	if description != "" {
		logging.Warning(description)

		// Descriptions are always communicated to the user
		print.Error(description)
	}
}

// Dirty hack to check failure type without importing the failure package
// We're getting rid of failures, not going to dance around to do this cleaner
func (e *Failure) IsFailure() {
}

// InputError tells us whether this is a user input error or not
func (e *Failure) InputError() bool {
	if e == nil {
		return false
	}
	return e.Type.User
}

// Type returns a FailureType that can be used to create your own failure types
func Type(name string, parents ...*FailureType) *FailureType {
	user := false
	for _, typ := range parents {
		if typ.Matches(FailUser) {
			user = true
		}
	}

	guid := uuid.New()
	return &FailureType{guid.String(), name, user, parents}
}

// Handle is what controllers would call to handle an error message, this will take care of calling the underlying
// handle method or logging the error if this isnt a Failure type
//
// If description is empty, only the error message is printed
func Handle(err error, description string) {
	handled = err
	switch t := err.(type) {
	case *Failure:
		t.Handle(description)
		return
	default:
		failure := failLegacy.New(err.Error())
		if description == "" {
			description = "Unknown Error:"
		}
		failure.Handle(description)
	}
}

// Handled returns the last handled error
func Handled() error {
	return handled
}

// ResetHandled resets handled to nil
func ResetHandled() {
	handled = nil
}

// ToError converts a failure to an error
func ToError(err error) error {
	switch v := err.(type) {
	case *Failure:
		err = v.ToError()
	}
	return err
}

// IsFailure returns whether the given error is of the Failure type
func IsFailure(err error) bool {
	switch t := err.(type) {
	case *Failure:
		_ = t // have to use t cause Golang
		return true
	default:
		return false
	}
}

// IsType is a little helper method for checking whether the error is of the given type
func IsType(err interface{}, typ interface{}) bool {
	return reflect.TypeOf(err) == reflect.TypeOf(typ)
}

// Recover is a helper function to use for catching panic
func Recover() {
	if r := recover(); r != nil {
		logging.Warning("Recovered from panic: %v", r)
	}
}

// trace returns the calling file and line
func trace() (string, int) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		logging.Debug("Could not get caller for logging message")
	}

	c := 1
	for strings.HasSuffix(file, "failures.go") {
		c++
		_, file, line, ok = runtime.Caller(c)
		if !ok {
			logging.Debug("Could not get caller for logging message")
			break
		}
	}

	return file, line
}
