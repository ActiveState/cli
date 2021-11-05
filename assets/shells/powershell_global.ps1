# {{.Start}}
{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
$env:{{$K}}="{{$V}}:$env:PATH"
{{- else}}
$env:{{$K}}="{{$V}}"
{{- end}}
{{- end}}
# {{.Stop}}