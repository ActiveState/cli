#!/bin/sh

{{range .denote}}
# {{.}}
{{end}}

{{.state}} exec --path {{.projectPath}} -- {{.exe}} "$@"
