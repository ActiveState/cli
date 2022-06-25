@echo off

{{range .denote}}
REM {{.}}
{{end}}

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
set {{$K}}={{$V}};%PATH%
{{- else}}
set {{$K}}={{$V}}
{{- end}}
{{- end}}

"{{.stateExec}}" "{{.stateSock}}" "{{.target}}" %*
