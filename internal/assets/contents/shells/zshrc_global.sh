{{if and (ne .Project "") (not .PreservePs1)}}
export PS1="[{{.Project}}] $PS1"
{{- end}}

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
export {{$K}}="{{$V}}:$PATH"
{{- else}}
export {{$K}}="{{$V}}"
{{- end}}
{{- end}}