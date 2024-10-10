cd "{{.WD}}"

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
$env:PATH = "{{ escapePwsh $V}};$env:PATH"
{{- else}}
$env:{{$K}} = "{{ escapePwsh $V }}"
{{- end}}
{{- end}}

{{ if .ExecAlias }}
New-Alias {{.ExecAlias}} {{.ExecName}}
{{ end }}

{{range $K, $CMD := .Scripts}}
function {{$K}} {
    & {{$.ExecName}} run {{$CMD}} $args
}
{{end}}

# Reset execution policy, since we had to set it to bypass to run this script
Set-ExecutionPolicy -Scope Process -ExecutionPolicy (Get-ExecutionPolicy -Scope User)

echo "{{ escapePwsh .ActivatedMessage}}"
echo "Warning: PowerShell is not yet officially supported."

{{.UserScripts}}
