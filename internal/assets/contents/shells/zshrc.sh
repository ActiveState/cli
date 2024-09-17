if [ -f $ZDOTDIR/.zshrc ]; then source $ZDOTDIR/.zshrc; fi

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
alias {{$K}}='{{$.ExecName}} run {{$CMD}}'
{{end}}

echo "{{.ActivatedMessage}}"

{{.UserScripts}}

