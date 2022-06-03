#!/bin/sh

{{range .denote}}
# {{.}}
{{end}}

"{{.stateExec}}" "{{.stateSock}}" "{{.targetPath}}/{{.exe}}" "$@"
