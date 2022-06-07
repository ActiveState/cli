#!/bin/sh

{{range .denote}}
# {{.}}
{{end}}

"{{.stateExec}}" "{{.stateSock}}" "{{.target}}" "$@"
