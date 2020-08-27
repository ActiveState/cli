@echo off
SET PROMPT=[{{.Owner}}/{{.Name}}]$S$P$G

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
set {{$K}}={{$V}};%PATH%
{{- else}}
set {{$K}}={{$V}}
{{- end}}
{{- end}}

{{$execCmd := .ExecName}}

{{ if .ExecAlias }}
{{$execCmd = .ExecAlias}}
DOSKEY {{.ExecName}}="{{.ExecAlias}}" $*
{{ end }}

{{range $K, $CMD := .Scripts}}
DOSKEY {{$K}}="{{$execCmd}}" run "{{$CMD}}" $*
{{end}}

cd {{.WD}}

{{.UserScripts}}
