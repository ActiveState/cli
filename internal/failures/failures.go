package failures

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/rs/xid"
)

var (
	// FailUser identifies a failure as a user facing failure, this doesn't use the Type method as the Type method does
	// not support setting the user bool. This should be the ONLY failure type that does this.
	FailUser = &FailureType{xid.New().String(), "failures.fail.user", true, []*FailureType{}}

	// failLegacy identifies a failure as a legacy failure, this is for internal use only
	failLegacy = Type("failures.fail.legacy")

	// FailIO identifies a failure as an IO failure
	FailIO = Type("failures.fail.io")

	// FailOS identifies a failure as an OS failure
	FailOS = Type("failures.fail.os")

	// FailInput identifies a failure as an input failure
	FailInput = Type("failures.fail.input", FailUser)

	// FailUserInput identifies a failure as an input failure
	FailUserInput = Type("failures.fail.userinput", FailInput, FailUser)

	// FailCmd identifies a failure as originating from the command that was ran
	FailCmd = Type("failures.fail.cmd", FailUser)

	// FailRuntime identifies a failure as a runtime failure (mainly intended for calls to the runtime package)
	FailRuntime = Type("failures.fail.runtime")

	// FailVerify identifies a failure as being due to the failure toverify something
	FailVerify = Type("failures.fail.verify")

	// FailNotFound identifies a failure as being due to an item not being found
	FailNotFound = Type("failures.fail.notfound")

	// FailNetwork identifies a failure due to a networking issue
	FailNetwork = Type("failures.fail.network")

	// FailArchiving identifies a failure due to archiving (compressing or decompressing)
	FailArchiving = Type("failures.fail.archiving")

	// FailMarshal identifies a failure due to marshalling or unmarshalling
	FailMarshal = Type("failures.fail.marshal")

	// FailThirdParty identifies a failure due to a third party component (ie. we cannot infer the real reason)
	FailThirdParty = Type("failures.fail.thirdparty")
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

	_, file, line, ok := runtime.Caller(1)
	if !ok {
		logging.Debug("Could not get caller for logging message")
	}

	if strings.HasSuffix(file, "failures.go") {
		_, file, line, ok = runtime.Caller(2)
		if !ok {
			logging.Debug("Could not get caller for logging message")
		}
	}

	logging.Debug("Failure '%s' created: %s (%v). File: %s, Line: %d", f.Name, message, params, file, line)
	return &Failure{locale.T(message, input), f, file, line}
}

// Wrap wraps another error
func (f *FailureType) Wrap(err error) *Failure {
	logging.Debug("Failure '%s' wrapped: %v", f.Name, err)
	return f.New(err.Error())
}

// Failure holds an actual failure, do not call this directly, use Fail and UserFail instead
type Failure struct {
	Message string
	Type    *FailureType
	File    string
	Line    int
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
	return errors.New(e.Error())
}

// Log the failure
func (e *Failure) Log() {
	logging.Error(fmt.Sprintf("%s: %s", e.Type.Name, e.Message))
}

// Handle handles the error message, this is used to communicate that the error occurred in whatever fashion is
// most relevant to the current error type
func (e *Failure) Handle(description string) {
	if description != "" {
		logging.Warning(description)

		// Descriptions are always communicated to the user
		print.Error(description)
	}

	e.Log()

	print.Error(e.Error())
}

// Type returns a FailureType that can be used to create your own failure types
func Type(name string, parents ...*FailureType) *FailureType {
	pc, file, line, ok := runtime.Caller(1)
	fun := runtime.FuncForPC(pc)

	if !ok {
		// This shouldn't ever happen to my knowledge, unless this function were a main function there will always be one
		// caller up the chain
		panic("runtime.Caller(1) failing in failures.Type")
	}

	pkg := strings.Split(filepath.Base(fun.Name()), ".")[0]
	if !strings.HasPrefix(name+".fail.", pkg) {
		panic(fmt.Sprintf("Invalid type name: %s, it should be in the format of `%s.fail.<name>`. Called from: %s:%d (%s)", name, pkg, file, line, fun.Name()))
	}

	user := false
	for _, typ := range parents {
		if typ.Matches(FailUser) {
			user = true
		}
	}

	guid := xid.New()
	return &FailureType{guid.String(), name, user, parents}
}

// Handle is what controllers would call to handle an error message, this will take care of calling the underlying
// handle method or logging the error if this isnt a Failure type
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
