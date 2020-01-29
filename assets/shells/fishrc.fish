function fish_prompt
    echo "[{{.Owner}}/{{.Name}}] % "
end

{{range $K, $V := .Env}}
set -xg {{$K}} "{{$V}}"
{{end}}

{{range $K, $CMD := .Scripts}}
alias {{$K}}='state run {{$CMD}}'
{{end}}

cd "{{.WD}}"

{{.UserScripts}}
