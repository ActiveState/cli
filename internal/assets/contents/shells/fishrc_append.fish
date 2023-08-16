# {{.Start}}
{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
set -xg {{$K}} "{{$V}}:$PATH"
{{- else}}
set -xg {{$K}} "{{$V}}"
{{- end}}
{{- end}}
if test ! -z "${{.ActivatedEnv}}"
  and test -f "${{.ActivatedEnv}}/{{.ConfigFile}}"
  echo "State Tool is operating on project ${{.ActivatedNamespaceEnv}}, located at ${{.ActivatedEnv}}"
end
# {{.Stop}}
