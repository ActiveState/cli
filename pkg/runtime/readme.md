# Runtime Package

The runtime package is responsible for sourcing a runtime based on a provided buildplan, as well as for providing
insights into that sourced runtime.

## Design Goals

A fundamental goal of the runtime package (and really any package) is that it is intuitive to maintain. Meaning when we
don't touch this code for 6 months and then come back to it we can still easily tell what's going on.

The main method of achieving this goal is by minimizing the responsibilities of the runtime package. By having it no be
aware of projects, buildscripts, analytics, etc. we facilitate a much cleaner boilerplate that is easier to grok than
if it were dealing with all these concepts.

Additionally we keep our use of channels very minimal, and centralize their use in key location so as to avoid passing
channels between functions or layers of logic.

As we further grow this runtime package we may find that certain responsibilities start to obfuscate the core logic
again, we should remain sensitive to this, removing responsibilities and shifting it into other standalone packages is
always an option.

### Avoid Dependencies

The runtime package should itself have no awareness of projects, buildscripts, or anything else not absolutely vital
for the purpose of installing a runtime.

Note we do provide project information for annotation purposes, because executors rely on it. Over time we should try
and remove executors from the runtime package, because it's really not part of sourcing a functional runtime, it's
more of a distinct post-processing step.

## Responsibilities

Anything not covered under these responsibilities should not be introduced into the runtime package without a good
reason and discussion with the tech-lead.

- Sourcing a runtime based on a provided buildplan
- Providing insights into the sourced runtime
- Handle sharing of sourced artifacts between multiple runtimes
    - Specifically this is handled by the "depot"
- Firing of events through the events package

### Sub-Packages

Any responsibilities provided by these sub-packages should NOT be handled anywhere else.

- events
    - Provide event hooks for interactions with runtime processes
        - eg. for progress indication or analytics
        - Note this is handled through the `events` sub-package.
- internal/buildlog
    - Interact with buildlog streamer
        - ie. provide progress information on in-progress builds
    - Firing of events through the events package
- internal/camel
    - Facilitate sourcing of camel runtimes
    - It does this by pre-processing a camel artifact and injecting a runtime.json that alternate builds normally
      produce
- internal/envdef
    - Facilitate reading of runtime.json files, and merging multiple runtime.json files together.

