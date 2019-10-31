The "expect" package is based on the github.com/Netflix/go-expect code.

The source code is licensed under the [Apache 2 license](https://github.com/Netflix/go-expect/blob/master/LICENSE).

Changes have been made to

- `console.go`: We are now using the abstract pseudo terminal for both Windows and Linux.
- `passthrough_pipe.go`: The original implementation was not working on Windows.  This has been fixed now.
