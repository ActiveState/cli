@echo off
SET PROMPT=[{{.Owner}}/{{.Name}}]$S$P$G

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
set {{$K}}={{$V}};%PATH%
{{- else}}
set {{$K}}={{$V}}
{{- end}}
{{- end}}

{{range $K, $CMD := .Scripts}}
DOSKEY {{$K}}=state run "{{$CMD}}" $*
{{end}}

{{ if .ExecAlias }}
DOSKEY state="{{.ExecAlias}}" $*
{{ end }}

cd {{.WD}}

{{.UserScripts}}
