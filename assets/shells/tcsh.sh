# We don't need to source the rcfile here (like the other shells do)  because
# the mechanism we are using to spawn the subshell is already loading it.
#
# Also, it's ineffectual to attempt setting a prompt in this script since those
# `set` variables are not inherited when we spawn the sub shell via exec.  Only
# `setenv` values are inherited.

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
setenv {{$K}} "{{$V}}:$PATH"
{{- else}}
{{- end}}
{{- end}}

{{range $K, $CMD := .Scripts}}
alias {{$K}} '{{.Exec}} run {{$CMD}}'
{{end}}

{{ if .ExecAlias }}
alias state='{{.ExecAlias}}'
{{ end }}

cd "{{.WD}}"

{{.UserScripts}}
