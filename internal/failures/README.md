Failures are our versions of errors and are mainly used to differentiate between errors that are user facing and errors
that are app facing (ie. us).

User errors are printed, app errors are logged. You define errors by calling `failures.App.New(..)` or `failures.User.New(..)`
just like you would call `errors.New(..)`. Then in the controller you use `failures.Handle(err, "Description of context")` 
for the error to be "handled", meaning the error is communicated to the relevant party (stdout, or the log).