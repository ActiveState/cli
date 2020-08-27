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

{{range $K, $CMD := .Scripts}}
alias {{$K}}='{{.Exec}} run {{$CMD}}'
{{end}}

{{ if .ExecAlias }}
alias state='{{.ExecAlias}}'
{{ end }}

cd "{{.WD}}"

{{.UserScripts}}
