{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
setenv {{$K}} "{{$V}}:$PATH"
{{- else}}
setenv {{$K}} "{{$V}}"
{{- end}}
{{- end}}
