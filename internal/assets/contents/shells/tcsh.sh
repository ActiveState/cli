# We don't need to source the rcfile here (like the other shells do)  because
# the mechanism we are using to spawn the subshell is already loading it.
#
# Also, it's ineffectual to attempt setting a prompt in this script since those
# `set` variables are not inherited when we spawn the sub shell via exec.  Only
# `setenv` values are inherited.

if ( -f ~/.cshrc ) source ~/.cshrc

# Other shells have the capability to pass an rc file flag when starting a shell.
# However, tcsh does not have that capability and will always source ~/.tcshrc.
# As a result, we have to do `tcsh -c source tcsh.sh ; exec tcsh`, the latter of which sources
# ~/.tcshrc.
# This will cause a redundant "State Tool is operating on project ..., located at ..." message
# since the necessary trigger variables are defined in this tcsh.sh script, which is loaded first.
# So, define a one-time variable that ~/.tcshrc file uses to prevent printing the redundant message.
# Subsequent tcsh invocations will not have this variable defined and will print the message as
# expected.
setenv ACTIVESTATE_TCSH_FIRST_RUN 1

cd "{{.WD}}"

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
setenv {{$K}} "{{$V}}:$PATH"
{{- else}}
setenv {{$K}} "{{$V}}"
{{- end}}
{{- end}}

{{ if .ExecAlias }}
alias {{.ExecName}}='{{.ExecAlias}}'
{{ end }}

{{range $K, $CMD := .Scripts}}
alias {{$K}} '{{$.ExecName}} run {{$CMD}}'
{{end}}

echo "{{.ActivatedMessage}}"

{{.UserScripts}}
