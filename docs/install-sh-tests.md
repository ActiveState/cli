# Test runs for install.sh script

This is a list of docker commands to test some expected behavior of the `install.sh` script.

## Install and activate

Install State Tool and activate project `ActiveState/cli` afterwards.

```sh
docker run --rm -it -v $PWD/installers:/installers -w /root buildpack-deps:bionic-curl \
    /installers/install.sh --activate ActiveState/cli -t /usr/local/bin
```

### User interaction

Confirm all defaults

### Expected behavior

You should end up in a shell with an activated state.

## Install and try to activate, but PATH is not set

Install State Tool and try to activate project `ActiveState/cli` but it does
not work, because the State Tool is not installed in a PATH directory.

```sh
docker run --rm -it -v $PWD/installers:/installers -w /root buildpack-deps:bionic-curl \
    /installers/install.sh --activate ActiveState/cli -t /root/.local/bin
```

### User interaction

Confirm all defaults

### Expected behavior

- The State Tool gets installed under `/root/.local/bin`.
- You see an error message that the State Tool could not be activated.
- You see instructions how to install the State Tool and then activate the project manually or with the install script.

## Invalid options 1

You cannot run the install script without prompts, when activating a project.

```sh
docker run --rm -it -v $PWD/installers:/installers -w /root buildpack-deps:bionic-curl \
    /installers/install.sh --activate ActiveState/cli -n
```

### Expected behavior

You see an error message that `-n` and `--activate` cannot be used at the same time.

## Invalid options 2

You always have to specify `-n` when specifying `-f`

```sh
docker run --rm -it -v $PWD/installers:/installers -w /root buildpack-deps:bionic-curl \
    /installers/install.sh -f
```

### Expected behavior

You see an error message that `-f` requires `-n`.

## Custom State Tool name

Install State Tool and activate project `ActiveState/cli` afterwards.
Overwrite the name of the State Tool to `as`

```sh
docker run --rm -it -v $PWD/installers:/installers -w /root buildpack-deps:bionic-curl \
    /installers/install.sh --activate ActiveState/cli -t /usr/local/bin -e as
```

### User interaction

Confirm all defaults.

### Expected behavior

You should end up in a shell with an activated state.

## No prompt

Install the State Tool with no prompts

```sh
docker run --rm -it -v $PWD/installers:/installers -w /root buildpack-deps:bionic-curl \
    /installers/install.sh -n
```

### Expected behavior

Should install to a directory on your path

You should see a message saying `You may now start using the 'state' program`

## No prompt with target not in PATH

Install the State Tool with no prompts and a target not in the current PATH

```sh
docker run --rm -it -v $PWD/installers:/installers -w /root buildpack-deps:bionic-curl \
    /installers/install.sh -n -t /root/.local/bin
```

### Expected behavior

Should install to the provided directory

You should see a message instructing you to source your RC file.

## Previous installation detected

Install the State Tool with defaults and then attempt to install again

```sh
docker run --rm -it -v $PWD/installers:/installers -w /root buildpack-deps:bionic-curl
```

From inside the docker container

```sh
/installers/install.sh
```

Run above command again

```sh
/installers/install.sh
```

### User interaction

Confirm all defaults

### Expected behavior

When installing for the second time you should be presented with a message
stating:

```sh
Previous installation detected at <installation-path>
To update the State Tool to the latest version, please run 'state update'.
To install in a different location, please specify the installation directory with '-t TARGET_DIR'.
```

The State Tool artifact was **NOT** downloaded.

### Follow up 1

Run in the same docker container

```sh
/installers/install.sh -n -f
```

#### Expected behavior

When installing, it should warn the user that it is overwriting an existing solution.

### Follow up 2

Run in the same docker container

```sh
/installers/install.sh -t /opt/state
```

#### Expected behavior

State Tool should install into /opt/state
