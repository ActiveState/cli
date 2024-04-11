if [ -f ~/.bashrc ]; then source ~/.bashrc; fi

{{if and (ne .Owner "") (not .PreservePs1) }}
export PS1="[{{.Owner}}/{{.Name}}] $PS1"
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
