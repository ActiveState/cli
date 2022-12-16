@echo off
{{- if ne .Project ""}}
SET PROMPT=[{{.Project}}]$S$P$G
{{- end}}

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
set {{$K}}={{$V}};%PATH%
{{- else}}
set {{$K}}={{$V}}
{{- end}}
{{- end}}