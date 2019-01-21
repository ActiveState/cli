@echo off
SET PROMPT=[{{.Owner}}\\{{.Name}}]$S$P$G

{{range $K, $V := .Env}}
set {{$K}}={{$V}}
{{end}}

{{range $K, $CMD := .Commands}}
DOSKEY {{$K}}="state run {{$CMD}}"
{{end}}

cd {{.WD}}

{{.UserScripts}}