# relay

    go get github.com/codemodus/relay

Package relay provides a simple mechanism for relaying control flow based upon
whether a checked error is nil or not.

## Usage

```go
func CodedFns(handler ...func(error)) (CodedCheckFunc, CodedTripFunc)
func DefaultHandler() func(error)
func Fns(handler ...func(error)) (CheckFunc, TripFunc)
func Handle()
type CheckFunc
type CodedCheckFunc
type CodedError
    func (ce *CodedError) Error() string
    func (ce *CodedError) ExitCode() int
type CodedTripFunc
    func CodedTripFn(ck CodedCheckFunc) CodedTripFunc
type ExitCoder
type Relay
    func New(handler ...func(error)) *Relay
    func (r *Relay) Check(err error)
    func (r *Relay) CodedCheck(code int, err error)
type TripFunc
    func TripFn(ck CheckFunc) TripFunc
```

### Setup

```go
import (
    "github.com/codemodus/relay"
)

func main() {
    r := relay.New()
    defer relay.Handle()

    err := fail()
    r.Check(err)

    // prints "{cmd_name}: {err_msg}" to stderr
    // calls os.Exit with code set as 1
}
```

### Setup (Custom Handler)

```go
    h := func(err error) {
        fmt.Println(err)
        fmt.Println("extra message")
    }

    r := relay.New(h)
    defer relay.Handle()

    err := fail()
    r.Check(err)

    fmt.Println("should not print")

    // Output:
    // always fails
    // extra message
```

### Setup (Eased Usage)

```go
    ck := relay.New().Check
    defer relay.Handle()

    err := fail()
    ck(err)

    // prints "{cmd_name}: {err_msg}" to stderr
    // calls os.Exit with code set as 1
```

### Setup (Coded Check)

```go
    ck := relay.New().CodedCheck
    defer relay.Handle()

    err := fail()
    ck(3, err)

    // prints "{cmd_name}: {err_msg}" to stderr
    // calls os.Exit with code set as first arg to r.CodedCheck ("ck")
```

### Setup (Trip Function)

```go
    ck := relay.New().Check
    trip := relay.TripFn(ck)
    defer relay.Handle()

    n := three()
    if n != 2 {
        trip("must receive %v: %v is invalid", 2, n)
    }

    fmt.Println("should not print")

    // prints "{cmd_name}: {trip_msg}" to stderr
    // calls os.Exit with code set as 1
```

### Setup (Eased Usage - Check and Trip)

```go
    ck, trip := relay.Fns()
    defer relay.Handle()

    err := mightFail()
    ck(err)

    n := three()
    if n != 2 {
        trip("must receive %v: %v is invalid", 2, n)
    }

    fmt.Println("should not print")

    // prints "{cmd_name}: {trip_msg}" to stderr
    // calls os.Exit with code set as 1
```

## More Info

### Background

https://github.com/golang/go/issues/32437#issuecomment-510214015

## Documentation

View the [GoDoc](https://pkg.go.dev/github.com/codemodus/relay)

## Benchmarks

N/A
