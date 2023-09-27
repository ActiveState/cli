# {{.Start}}
{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
setenv {{$K}} "{{$V}}:$PATH"
{{- else}}
{{- end}}
{{- end}}
if ( "${{.ActivatedEnv}}" != "" && -f "${{.ActivatedEnv}}/{{.ConfigFile}}" ) then
  echo "State Tool is operating on project ${{.ActivatedNamespaceEnv}}, located at ${{.ActivatedEnv}}"
endif
# {{.Stop}}
