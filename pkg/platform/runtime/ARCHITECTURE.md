# Improved runtime architecture

Package runtime provides functions to setup and use a runtime downloaded from
the ActiveState Platform.

## Usage

The general usage pattern is as follows:

	rt, err := runtime.New(target)
	if err != nil {
		if !runtime.IsNeedsUpdateError(err) {
			return err
		}
		if err = rt.Update(messageHandler); err != nil {
			return err
		}
	}

	env, err = r.Environ(true, projectDir)
	if err != nil {...}

## Package structure

The runtime package consists of the following sub-packages:

	pkg/platform/runtime
	├── artifact
	├── envdef
	├── model
	├── setup
	│   ├── buildlog
	│   ├── events
	│   └── implementations
	│       ├── alternative
	│       └── camel
	└── store

### Toplevel package

The toplevel package `runtime` comprises functions to set up a runtime that
is already installed locally.

**Invariant**:

- No communication to the API backend is performed in this package

### Artifact package

The `artifact` package provides an abstraction of artifact information that
can be generated from recipes or the build status response. The idea is to
address the use-case where we want to meta-data like the dependency tree
about the current project.

### Model package

The package `model` implements tightly scoped methods to communicate with the
Platform API.  The model implementation should be simple to mock to support unit tests.

### Envdef package


The package `envdef` implements methods to parse, merge and apply environment definitions that are shipped with artifact files.

### Setup package

The setup package provides functionality to actually install / set up a
runtime locally.  The main struct is called `setup.Setup`.

**Invariants**:

- It is the only package where the `model` package is used.
- When `setup/Setup.Update()` finishes successfully, the runtime can
be loaded from the disk without further Platform communication.
- This package does not comprise build engine specific code. It is hidden
behind the `setup/implementations` interface

### Setup.Events package

The `events` package in the `setup` directory provides structs to handle setup events, which can be sent from parallel running threads.  The `RuntimeEventHandler` translates these events to commands for "digester" implementations.  The default digesters are implemented in the `runbits.changesummary` and `runbits.progressbar` packages.

### Runtime implementations

As we have two (maybe more) flavors of builds (Camel and Alternative), we split out the specific implementations for how to set them up in an implementation package called `implementations`.

The actual runtime implementations are in the sub-packages `alternative` and `camel`.

**Invariant**:

- The functions in these package are not calling any model functions.
