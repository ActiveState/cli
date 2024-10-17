{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
$env:{{$K}} = "{{ escapePwsh $V }};$env:PATH"
{{- else}}
$env:{{$K}} = "{{ escapePwsh $V }}"
{{- end}}
{{- end}}
