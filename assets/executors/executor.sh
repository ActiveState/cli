#!/bin/sh

{{range .denote}}
# {{.}}
{{end}}

{{.state}} exec --path {{.targetPath}} -- {{.exe}} "$@"
