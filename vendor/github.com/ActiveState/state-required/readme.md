Add this Go package to your code to ensure developers use the ActiveState "state" tool to run their Go project.

This only limits code ran using `go run`, it will cancel out for any compiled code.

This is for internal dogfooding only.

## Installation

```
go get -u github.com/ActiveState/state-required
```

## Usage

Add the following import statement to your main package

```
import (_ "github.com/ActiveState/state-required/require")
```