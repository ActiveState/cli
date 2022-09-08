{{if ne .Project ""}}
__state_prompt() {
  echo "[{{.Project}}]"
}
if [ "$PROMPT_COMMAND" != "" ]; then
  PROMPT_COMMAND+=" "
fi
PROMPT_COMMAND+='__state_prompt'
{{end}}

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
export {{$K}}="{{$V}}:$PATH"
{{- else}}
export {{$K}}="{{$V}}"
{{- end}}
{{- end}}
