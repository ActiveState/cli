@echo off
SET PROMPT=[{{.Owner}}\\{{.Name}}]$S$P$G

{{range $K, $V := .Env}}
set {{$K}}={{$V}}
{{end}}

cd {{.WD}}