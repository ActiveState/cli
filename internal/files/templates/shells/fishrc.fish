echo "ActiveState"

{{range $K, $V := .Env}}
set -x {{$K}} "{{$V}}"
{{end}}
cd "{{.WD}}"
