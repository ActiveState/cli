#!/bin/sh

{{range .denote}}
# {{.}}
{{end}}

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
export {{$K}}="{{$V}}:$PATH"
{{- else}}
export {{$K}}="{{$V}}"
{{- end}}
{{- end}}

"{{.stateExec}}" "{{.stateSock}}" "{{.targetFile}}" "{{.nameSpace}}" "{{.commitID}}" "{{.headless}}" "$@"
