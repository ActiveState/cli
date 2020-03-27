# {{.Start}}

{{range $K, $V := .Env}}
setenv {{$K}} "{{$V}}"
{{end}}

# {{.Stop}}