if [ -f ~/.bashrc ]; then source ~/.bashrc; fi

{{if ne .Owner ""}}
__state_prompt() {
  echo "[{{.Owner}}/{{.Name}}]"
}
if [ "$PROMPT_COMMAND" != "" ]; then
  PROMPT_COMMAND+=" "
fi
PROMPT_COMMAND+='__state_prompt'
{{end}}

cd "{{.WD}}"

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

echo "{{.ActivatedMessage}}"

{{.UserScripts}}
