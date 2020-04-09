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
alias {{$K}}='state run {{$CMD}}'
{{end}}

cd "{{.WD}}"

{{.UserScripts}}
