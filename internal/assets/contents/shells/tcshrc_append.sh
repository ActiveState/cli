# {{.Start}}
{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
setenv {{$K}} "{{$V}}:$PATH"
{{- else}}
setenv {{$K}} "{{$V}}"
{{- end}}
{{- end}}
{{- if .Default }}
if ( $?{{.ActivatedEnv}} ) then
  if ( -f "${{.ActivatedEnv}}/{{.ConfigFile}}" ) then
    echo "State Tool is operating on project ${{.ActivatedNamespaceEnv}}, located at ${{.ActivatedEnv}}"
  endif
endif
{{- end}}
# {{.Stop}}
