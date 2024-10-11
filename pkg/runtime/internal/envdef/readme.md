Package envdef implements a parser for the runtime environment for alternative builds

Builds that are built with the alternative build environment, include runtime.json files that define which environment
variables need to be set to install and use the provided artifacts.
The schema of this file can be downloaded [here](https://drive.google.com/drive/u/0/my-drive)

The same parser and interpreter also exists
in [TheHomeRepot](https://github.com/ActiveState/TheHomeRepot/blob/master/service/build-wrapper/wrapper/runtime.py)

Changes to the runtime environment definition schema should be synchronized between these two places. For now, this can
be most easily accomplished by keeping the description of test cases in
the [cli repo](https://github.com/ActiveState/cli/blob/master/pkg/platform/runtime/envdef/runtime_test_cases.json)
and [TheHomeRepot](https://github.com/ActiveState/TheHomeRepot/blob/master/service/build-wrapper/runtime_test_cases.json)
in sync.

Examples:

## Define a PATH and LD_LIBRARY_PATH variable

Assuming the runtime is installed to a directory `/home/user/.cache/installdir`, the following definition asks to set
the PATH variables to`/home/user/.cache/installdir/bin:/home/user/.cache/installdir/usr/bin` and`LD_LIBRARY_PATH`
to`/home/user/.cache/installdir/lib`The set `inherit` flag on the `PATH` variable ensures that the `PATH` value is
prepended to the existing `PATH` that is already set in the environment.

```json
{
  "env": [
    {
      "env_name": "PATH",
      "values": [
        "${INSTALLDIR}/bin",
        "${INSTALLDIR}/usr/bin"
      ],
      "join": "prepend",
      "inherit": true,
      "separator": ":"
    },
    {
      "env_name": "LD_LIBRARY_PATH",
      "values": [
        "${INSTALLDIR}/lib"
      ],
      "join": "prepend",
      "inherit": false,
      "separator": ":"
    }
  ],
  "installdir": "installdir"
}
```

The installdir is used during the unpacking step to identify the directory inside the artifact tarball that needs to be
unpacked to `/home/user/.cache/installdir`

## Joining two definitions

Assume we have a second environment definition file exists with the following contents:

```json
{
  "env": [
    {
      "env_name": "PATH",
      "values": [
        "${INSTALLDIR}/bin",
        "${INSTALLDIR}/usr/local/bin"
      ],
      "join": "prepend",
      "inherit": true,
      "separator": ":"
    },
    {
      "env_name": "LD_LIBRARY_PATH",
      "values": [
        "${INSTALLDIR}/lib",
        "${INSTALLDIR}/lib64"
      ],
      "join": "prepend",
      "inherit": false,
      "separator": ":"
    }
  ],
  "installdir": "installdir"
}
```

Merging this environment definition into the previous one sets the `PATH`
to `/home/user/.cache/installdir/bin:/home/user/.cache/installdir/usr/local/bin:/home/user/.cache/installdir/usr/bin`.
Note, that duplicate values are filtered out. Likewise the `LD_LIBRARY_PATH` will end up
as `/home/user/.cache/installdir/lib:/home/user/.cache/installdir/lib64`

In this example, the values were joined by prepending the second definition to the first. Other join strategies
are `append` and `disallowed`.

The `disallowed` join strategy can be used if a variable should have only ONE value, and this value needs to be the same
or undefined between all artifacts
that depend on it.

## Usage

- Environment definition files can be parsed from a file with the `NewEnvironmentDefinition()` function.
- Two environment definitions `ed1` and `ed2` can be merged like so:
  ed1.Merge(ed2)
- Once the installation directory is specified, the variable values can be expanded:
  ed.ExpandVariables("/home/user/.cache/installdir")
