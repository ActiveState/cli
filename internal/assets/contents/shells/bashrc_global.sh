{{- if ne .Project ""}}
if [ -z "$PROMPT_COMMAND" ]; then
  export PS1="[{{.Project}}] $PS1"
fi
{{- end}}

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
export {{$K}}="{{$V}}:$PATH"
{{- else}}
export {{$K}}="{{$V}}"
{{- end}}
{{- end}}
