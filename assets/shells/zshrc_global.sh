export PS1="[{{.Project}}] $PS1"

precmd() { eval "$PROMPT_COMMAND" }

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
export {{$K}}="{{$V}}:$PATH"
{{- else}}
export {{$K}}="{{$V}}"
{{- end}}
{{- end}}