PyLog
====



A simple logging module that mimics the behavior of Python's logging module.

All it does basically is wrap Go's logger with nice multi-level logging calls, and
allows you to set the logging level of your app in runtime.

Logging is done just like calling fmt.Sprintf:

```go
logging.Info("This object is %s and that is %s", obj, that)
```

Logging level can be set to whatever you want it to be, in runtime. Contrary to Python that specifies a minimal level, this logger is set with a bit mask of active levels.

```go
//for INFO and ERROR use:
logging.SetLevel(logging.INFO | logging.ERROR)

// For everything but debug and info use:
logging.SetLevel(logging.ALL &^ (logging.INFO | logging.DEBUG))
```

As with the standard log, you can specify any `io.Writer` type interface and send the log's output to it instead of the default stderr.

### Installation:

```
go get github.com/dvirsky/go-pylog/logging
```

### Usage Example:

```go
package main

import (
	"github.com/dvirsky/go-pylog/logging"
)

func main() {

	logging.Info("All Your Base Are Belong to %s!", "us")

	logging.Critical("And now with a stack trace")
}
```

### Lazily Evaluated functions as arguments

You can give the logger a function with the signature `func() interface{}`, and it will only execute it if the message is being printed, and simply format its output into the log.

For example:
```go
// just pass a lambda
logging.Debug("The time now is %s", func() interface{} { return time.Now()})

// or for more complex stuff:

// let's say we have this heavy weight function we want to log, 
// but only if the relevant level is activ
func sumSeries(s []int) int {
    ret := 0
    for _, n := range s {
        ret += n
    }
    return ret
}

// Wrapping it in this lazy lambda this will execute the function only if the level matches Info
logging.Info("The sum of my series is %d", func() interface{} { return sumSeries(mySeries)})
```



### Custom Handlers

By default we just write to Go's log, and you can set the output stream of it. But you can add a custom `handler` that will receive the raw unformatted messages, format them and do whatever it wants with them.  The logger currently supports a single handler. 

This was added for the use case of `Scribe`, that needs to receive messages as a pair of category and message. So an io.Writer was not applicable.

The inreface for a `LogHandler` is:

```go
type LoggingHandler interface {
    Emit(level, file string, line int, message string, args ...interface{}) error
}
```

To set your own handler (or the provided scribe handler in the library), call `logging.SetHandler(myHandler)`.


### Custom Message formatting

It is possible to change the logger's display format. The default format is 
`"%[1]s @ %[2]s:%[2]d: %[4]s"` - resulting in messages looking like:
`INFO @ db.go:528: Registering plugin REPLICATION`. 

The indexes are there so you can change the order of the formatting elements if you want. [1] means the logging level, [2] is the file, [3] is the line and [4] is the unformatted message passed to the log.

To change the way they are formatted, call `logging.SetFormatString()`.


### Example Output:

```
2013/05/07 01:20:26 INFO @ db.go:528: Registering plugin REPLICATION
2013/05/07 01:20:26 INFO @ db.go:562: Registered 6 plugins and 22 commands
2013/05/07 01:20:26 INFO @ slave.go:277: Running replication watchdog loop!
2013/05/07 01:20:26 INFO @ redis.go:49: Redis adapter listening on 0.0.0.0:2000
2013/05/07 01:20:26 WARN @ main.go:69: Starting adapter...
2013/05/07 01:20:26 INFO @ db.go:966: Finished dump load. Loaded 2 objects from dump
2013/05/07 01:22:26 INFO @ db.go:329: Checking persistence... 0 changes since 2m0.000297531s
2013/05/07 01:22:26 DEBUG @ db.go:341: Sleeping for 2m0s
```
