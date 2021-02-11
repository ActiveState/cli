# Improved runtime architecture

Package runtime provides functions to setup and use a runtime downloaded from
the ActiveState Platform.

## Usage

The general usage pattern is as follows:

	setup, err := setup.NewSetup(proj, msgHandler)
	if err != nil {...}

	r, err = setup.InstalledRuntime()
	if runtime.IsNotInstalledError(err) {
		err = setup.InstallRuntime()
		if err != nil {...}
		r, err = setup.InstalledRuntime()
	}
	if err != nil {...}

	env, err = r.Environ()
	if err != nil {...}

## Package structure

The runtime package consists of the following sub-packages:

	pkg/platform/runtime2
	├── setup
	├── impl
	│   ├── alternative
	│   └── camel
	├── model
	│   └── client
	└── artifact

### Toplevel package

The toplevel package `runtime` comprises functions to set up a runtime that
is already installed locally.

**Invariant**:

- No communication to the API backend is performed in this package

### API package

The package `api` defines interfaces to all backend functions necessary to set
up a runtime locally.

Two implementations are provided in `api.client`: `Default` and `Mock` for testing

### Setup package

The setup package provides functionality to actually install / set up a
runtime locally.  The main struct is called `setup.Setup`.

**Invariants**:

- It is the only package where the `api` package is used.
- When `setup/Setup.InstallRuntime()` finishes successfully, the runtime can
be loaded from the disk without further Platform communication.
- This package does not comprise build engine specific code. It is hidden
behind the `setup/common/Setuper` interface

### Runtime implementations

As we have two (maybe more) flavors of builds (Camel and Alternative), we split out the specific implementations for how to set them up in an implementation package called `impl`.

The actual runtime implementations are in the sub-packages `alternative` and `camel`.

This `impl` package can contain common functionality between the specific implementations and defines the interfaces the implementations have to fulfill.

**Invariant**:

- The functions in these package are not calling any api functions.

Artifact package

The `artifact` package provides an abstraction of artifact information that
can be generated from recipes. The idea is to address the use-case where we
want to meta-data like the dependency tree about the current project.

## Tests

I suggest the following tests:

- setup/setup_test.go: tests the entire set up of a runtime based on a mocked
API client. This is the most complicated part, and it involves some
asynchronous operations. So, it will be nice to have some unit-tests
available, to test some edge cases, especially w.r.t. message handling.
- artifact/artifact_test.go: Tests to ensure that we can parse the returned
Recipe structure correctly.
- api/client/default_test.go: Here we could add some very focused integration
tests, that should fail if the backend changes in an incompatible way.
