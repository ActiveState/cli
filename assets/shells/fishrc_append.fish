# {{.Start}}

{{range $K, $V := .Env}}
set -xg {{$K}} "{{$V}}"
{{end}}

# {{.Stop}}