if [ -f $ZDOTDIR/.zshrc ]; then source $ZDOTDIR/.zshrc; fi

cd "{{.WD}}"

{{if ne .Owner ""}}
export PS1="[{{.Owner}}/{{.Name}}] $PS1"
{{end}}

precmd() { eval "$PROMPT_COMMAND" }

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
alias {{$K}}='{{$.ExecName}} run {{$CMD}}'
{{end}}

echo "{{.ActivateEventMessage}}"

{{.UserScripts}}

echo "{{.ActivatedMessage}}"
