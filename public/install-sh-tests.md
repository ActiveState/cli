# Test runs for install.sh script

This is a list of docker commands to test some expected behavior of the `install.sh` script.

## Install and activate

Install state tool and activate project `ActiveState/cli` afterwards.

```sh
docker run --rm -it -v $PWD/public:/scripts -w /root buildpack-deps:bionic-curl \
    /scripts/install.sh --activate ActiveState/cli -t /usr/local/bin
```

### User interaction

Confirm all defaults

### Expected behavior

You should end up in a shell with an activated state.

## Install and try to activate, but PATH is not set

Install state tool and try to activate project `ActiveState/cli` but it does
not work, because the state tool is not installed in a PATH directory.

```sh
docker run --rm -it -v $PWD/public:/scripts -w /root buildpack-deps:bionic-curl \
    /scripts/install.sh --activate ActiveState/cli -t /root/.local/bin
```

### User interaction

Confirm all defaults

### Expected behavior

- The state tool gets installed under `/root/.local/bin`.
- You see an error message that the state tool could not be activated.
- You see instructions how to install the state tool and then activate the project manually or with the install script.

## Invalid options

You cannot run the install script without prompts, when activating a project.

```sh
docker run --rm -it -v $PWD/public:/scripts -w /root buildpack-deps:bionic-curl \
    /scripts/install.sh --activate ActiveState/cli -n
```

### Expected behavior

You see an error message that `-n` and `--activate` cannot be used at the same time.


## Custom state tool name

Install state tool and activate project `ActiveState/cli` afterwards.
Overwrite the name of the state tool to `as`

```sh
docker run --rm -it -v $PWD/public:/scripts -w /root buildpack-deps:bionic-curl \
    /scripts/install.sh --activate ActiveState/cli -t /usr/local/bin -f as
```

### User interaction

Confirm all defaults.

### Expected behavior

You should end up in a shell with an activated state.
