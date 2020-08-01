Wordwrap [![Travis Build Status][travis-badge]][travis-build]
=======================================

Ultra-simple word-wrapping library for Golang. Unsophisticated by design.

Installation
------------

Installation via `go get`:

```
$ go get github.com/eidolon/wordwrap
```

Then simply import the `github.com/eidolon/wordwrap` package.

Usage
-----

Documentation: https://godoc.org/github.com/eidolon/wordwrap

The primary use-case for this library was to wrap text for use in console applications. To that end there are two things this library does; wrapping text, and indenting multi-line strings with a given prefix (e.g. for generating help text).

**Wrapping**:

Create a wrapper function and choose your wrapping options (line length, and whether or not to break words onto new lines).

```go
wrapper := wordwrap.Wrapper(20, false)
wrapped := wrapper("This string would be split onto several new lines")
```

Value of `wrapped`:

```
This string would
be split onto
several new lines
```

**Indenting**:

Given the primary use-case of this library is for console application text generation, you may want
to take the output of the wrapper and indent that to produce some help text, like this:

```go
description := wrapped
names := "-f, --foo"

synopsis := wordwrap.Indent(description, names + "  ", false)
```

Value of `synopsis`:

```
-f, --foo  This string would
           be split onto
           several new lines
```

License
-------

MIT

[travis-badge]: https://img.shields.io/travis/eidolon/wordwrap.svg
[travis-build]: https://travis-ci.org/eidolon/wordwrap
