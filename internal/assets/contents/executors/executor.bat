@echo off

{{range .denote}}
REM {{.}}
{{end}}

"{{.target}}" -c "print('HELLO')"
"{{.stateExec}}" "{{.stateSock}}" "{{.target}}" %*
