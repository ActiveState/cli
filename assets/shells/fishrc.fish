function fish_prompt
    echo "[{{.Owner}}/{{.Name}}] % "
end

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
set -xg {{$K}} "{{$V}}:$PATH"
{{- else}}
set -xg {{$K}} "{{$V}}"
{{- end}}
{{- end}}

{{ if .ExecAlias }}
alias {{.ExecName}}='{{.ExecAlias}}'
{{ end }}

{{range $K, $CMD := .Scripts}}
alias {{$K}}='{{$.ExecName}} run {{$CMD}}'
{{end}}

cd "{{.WD}}"

{{.UserScripts}}
