#!/bin/sh

# {{.denote}}

{{.exe}} shim --path {{.projectPath}} -- {{.command}} "$@"
