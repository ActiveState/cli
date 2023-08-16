# {{.Start}}
{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
export {{$K}}="{{$V}}:$PATH"
{{- else}}
export {{$K}}="{{$V}}"
{{- end}}
{{- end}}
if [[ ! -z "${{.ActivatedEnv}}" && -f "${{.ActivatedEnv}}/{{.ConfigFile}}" ]]; then
  echo "State Tool is operating on project ${{.ActivatedNamespaceEnv}}, located at ${{.ActivatedEnv}}"
fi
# {{.Stop}}
