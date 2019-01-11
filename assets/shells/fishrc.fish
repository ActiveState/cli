echo - Active state: {{.Project.Owner}}/{{.Project.Name}}

{{range $K, $V := .Env}}
set -xg {{$K}} "{{$V}}"
{{end}}

cd "{{.WD}}"
{{.UserScripts}}
