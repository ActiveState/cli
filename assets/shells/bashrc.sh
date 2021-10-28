if [ -f ~/.bashrc ]; then source ~/.bashrc; fi

{{ if .Owner }}
if [ -z "$PROMPT_COMMAND" ]; then
  export PS1="[{{.Owner}}/{{.Name}}] $PS1"
fi
{{end}}

{{ if .WD }}
cd "{{.WD}}"
{{ end }}

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
export {{$K}}="{{$V}}:$PATH"
{{- else}}
export {{$K}}="{{$V}}"
{{- end}}
{{- end}}

{{ if .ExecAlias }}
alias {{.ExecName}}='{{.ExecAlias}}'
{{ end }}

{{range $K, $CMD := .Scripts}}
function {{$K}} {
    {{$.ExecName}} run {{$CMD}} "$@"
}
export -f {{$K}}
{{end}}

{{ if .ActivatedMessage }}
echo "{{.ActivatedMessage}}"
{{ end }}

{{ if .UserScripts }}
{{.UserScripts}}
{{ end }}
cat ${BASH_SOURCE}
