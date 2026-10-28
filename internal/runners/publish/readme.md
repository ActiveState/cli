# Publishing Encrypted Private Python Ingredients

This guide walks through publishing a private Python package to the ActiveState Platform with `state publish --build`,
declaring its dependencies, and changing those dependencies later.

With `--build`, the State Tool packs your source directory into a wheel and encrypts it with your organization's key
before uploading it. The platform only sees the ingredient's name, namespace, version and dependencies, not your source
code. Projects in your organization decrypt and install it into their Python runtime's `site-packages`.

## Before You Start

- You are logged in: `state auth`.
- You are in a checkout of a project owned by your organization (it has an `activestate.yaml`). The project determines
  which organization's key encrypts the package.
- Your organization's key service is configured (`state config set privateingredient.key_service_url <url>`, plus the
  `privateingredient.*` credential settings your administrator gives you). Without it nothing is uploaded.

Throughout this guide, `<org>` is your organization name and `mypkg` is the package being published.

## 1. Lay Out the Source

Point `--build` at a directory whose children are the importable packages. A `pyproject.toml` beside them is optional,
but it lets the State Tool read the name, version, description and dependencies, so you don't have to pass them as flags.

```
mypkg-src/
├── pyproject.toml
└── mypkg/
    ├── __init__.py
    └── core.py
```

Only pure-Python packages are supported: compiled files (`.so`, `.pyd`, `.dll`, `.dylib`) are rejected. `__pycache__`
directories, `*.pyc` files and version-control directories are left out of the wheel.

## 2. Declare Dependencies

List them in `pyproject.toml` the same way you would for pip:

```toml
[project]
name = "mypkg"
version = "1.0.0"
description = "Internal helpers for the data team"
dependencies = [
  "requests>=2.31,<3",
  "six~=1.16",
  'pywin32>=300; sys_platform == "win32"',
  'tomli>=1; python_version < "3.11"',
]
```

Each entry becomes a runtime dependency on the platform's Python package of that name. Version ranges carry over, with
`~=1.16` becoming `>=1.16,<2` and `==4.2.*` becoming `>=4.2,<4.3`. An entry without a range, such as `"click"`, accepts
any version (`>=0`).

Platform and Python version markers become **conditions**, so the dependency is only installed where it applies:

| Marker                                                                        | Installed only on        |
|-------------------------------------------------------------------------------|--------------------------|
| `sys_platform == "win32"`, `platform_system == "Windows"`, `os_name == "nt"` | Windows                  |
| `sys_platform == "linux"`, `platform_system == "Linux"`                      | Linux                    |
| `sys_platform == "darwin"`, `platform_system == "Darwin"`                    | macOS                    |
| `python_version < "3.11"` (also `<=`, `>`, `>=`, `==`)                       | matching Python versions |
| `python_full_version >= "3.9.2"`                                             | matching Python versions |

Several of these can be combined with `and`, e.g. `sys_platform == "win32" and python_version >= "3.10"`.

Anything else is skipped with a notice when you publish: markers using `or`, parentheses or `!=`, other markers such as
`platform_machine`, optional dependencies (`extra == "..."`), and direct URL references (`pkg @ https://...`). Add a
skipped dependency by hand with `--depend` or a meta file (see step 5) if it is required.

## 3. Publish

From the project checkout:

```
state publish --build ./mypkg-src --namespace private/<org>/python
```

The namespace must be under `private/<org>`, the organization that owns the project. Before uploading, the State Tool
shows the ingredient it is about to publish and asks for confirmation:

```
Skipping pyproject.toml dependency 'colorama; platform_machine == "x86_64"' (unsupported environment marker
'platform_machine == "x86_64"'). Add it with '--depend' if it is required.

Prepared the following ingredient:

name: mypkg
namespace: private/<org>/python
version: 1.0.0
description: Internal helpers for the data team
dependencies:
    - dependency:
        name: requests
        namespace: language/python
        versionRequirements: '>=2.31,<3'
        type: runtime
    - dependency:
        name: pywin32
        namespace: language/python
        versionRequirements: '>=300'
        type: runtime
      conditions:
        - name: Windows
          namespace: kernel
          versionRequirements: '>=0'
    ...
    - dependency:
        name: private-builder
        namespace: builder
        versionRequirements: '>=0'

Do you want to publish this ingredient? Answer 'n' to edit it or save it to a file instead. (Y/n)
```

Check the dependency list here: it is exactly what will be published. The `private-builder` dependency is always added;
it is what decrypts and installs the package, so leave it in place.

If something is wrong, answer `n` and choose one of:

- **Edit it in my editor:** opens the meta data in `$EDITOR` (or the default program for `.yaml` files), then shows
  the result for confirmation again. Name, namespace and version can't be changed this way, because the package was
  already built with them: if you change them, your edits are saved to a meta file as below.
- **Save it to a meta file to edit and publish later:** writes `mypkg-1.0.0.meta.yaml` to the current directory and
  prints the command to publish it, e.g. `state publish --build ./mypkg-src --meta mypkg-1.0.0.meta.yaml`. Edit the file,
  then run that command. The meta file takes precedence over `pyproject.toml`, so its name, version and dependency list
  are used instead.
- **Cancel:** nothing is published.

Flags override `pyproject.toml`: `--name`, `--version` and `--description` replace the values in it, and any
`--depend`, `--depend-runtime`, `--depend-build` or `--depend-test` flag replaces its whole dependency list.

## 4. Install and Check

In any project of the same organization:

```
state install private/<org>/python:mypkg --ts now
state exec python3 -- -c "import mypkg"
```

`--ts now` makes the solver pick up the version you just published. The install summary lists the dependencies that
were pulled in, and conditional dependencies only appear on matching platforms.

The machine installing the package needs the same key service settings as the one that published it, because the
package is decrypted during installation.

## 5. Change the Dependencies of a Published Version

To change the dependencies of a version that is already published, export its meta file, edit it, and publish it back
as a new revision with `--edit`.

**Export the current meta file:**

```
state export ingredient-meta private/<org>/python/mypkg@1.0.0 > meta.yaml
```

Leave out `@1.0.0` to export the latest version. The file holds the ingredient's name, namespace, version and its
complete dependency list:

```yaml
name: mypkg
namespace: private/<org>/python
version: 1.0.0
description: ""
dependencies:
    - dependency:
        name: private-builder
        namespace: builder
        versionRequirements: '>=0'
        type: build
      conditions: []
    - dependency:
        name: requests
        namespace: language/python
        versionRequirements: '>=2.31,<3'
        type: runtime
      conditions: []
    ...
```

**Edit the `dependencies` list.** Change version ranges, remove entries, or add new ones. Each entry nests its fields
under `dependency:`, with optional `conditions:` beside it:

```yaml
    - dependency:
        name: urllib3
        namespace: language/python
        versionRequirements: '>=2'
        type: runtime
      conditions:
        - name: Linux
          namespace: kernel
          versionRequirements: '>=0'
```

For conditions, the kernel names are `Windows`, `Linux` and `Darwin` (namespace `kernel`). A Python version condition
is `name: python`, `namespace: language`, with a requirement such as `<3.11`.

**Publish the revision:**

```
state publish --edit --build ./mypkg-src --meta meta.yaml
```

The platform requires the source with every revision, so pass `--build` even when only the meta data changes.

Things to know about `--edit`:

- **The meta file replaces the whole dependency list.** Anything you delete from it is removed from the ingredient.
  Dependencies in `pyproject.toml` are not used when the meta file lists its own. If neither the meta file nor
  `pyproject.toml` lists any, the previous revision's dependencies are kept.
- **Name, namespace and version identify what you are editing; they cannot be changed.** To publish under a new
  version, see step 6.
- **Authors and description cannot be changed** on an existing version. The exported file leaves authors out and the
  description empty. If you fill the description in, the publish is refused with "Authors and description cannot be
  changed when editing an existing version."
- The `language/python` dependencies the platform adds by itself come back automatically, so it does no harm if they
  are missing from your file.

Run the export again afterwards to confirm the change.

## 6. Publish a New Version

Bump the version in `pyproject.toml` (or pass `--version`) and publish as in step 3, without `--edit`. A new version
takes its dependencies from `pyproject.toml`, flags and `--meta` as usual and does not inherit them from the previous
version.

Publishing a version that already exists without `--edit` is refused, so an existing version can't be overwritten by
accident.

## Troubleshooting

| Message                                                                 | What to do                                                                                     |
|-------------------------------------------------------------------------|------------------------------------------------------------------------------------------------|
| `Could not determine the ingredient name and version`                   | The message lists each missing field and why. Add it to `pyproject.toml`'s `[project]` table, pass `--name`/`--version`, or set it in a `--meta` file. A version listed in `[project].dynamic` can't be read and must be given this way. |
| `The '--build' flag requires a namespace under 'private/<org>'`         | Use a namespace under the organization that owns the project you are in.                      |
| `org key does not belong to this project's organization`                | The key service returned another organization's key. Check that you are in the right project. |
| `No organization key service is configured`                             | Configure the `privateingredient.*` settings (see Before You Start).                          |
| `Ingredient ... already exists. To edit an existing ingredient ...`     | Pass `--edit` to change an existing version, or bump the version to publish a new one.         |
| `Skipping pyproject.toml dependency ...`                                | The marker is not supported. Add the dependency with `--depend` or in a meta file.            |
