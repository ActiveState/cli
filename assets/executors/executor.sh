#!/bin/sh

{{range .denote}}
# {{.}}
{{end}}

{{.state}} shim --path {{.projectPath}} -- {{.exe}} "$@"
