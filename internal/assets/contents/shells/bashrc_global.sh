if [ -z "$PROMPT_COMMAND" ]; then
  export PS1="[{{.Project}}] $PS1"
fi

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
export {{$K}}="{{$V}}:$PATH"
{{- else}}
export {{$K}}="{{$V}}"
{{- end}}
{{- end}}
