@echo off

{{range .denote}}
REM {{.}}
{{end}}

"{{.stateExec}}" "{{.stateSock}}" "{{.target}}" %*
