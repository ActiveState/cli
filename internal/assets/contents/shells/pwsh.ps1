{{if and (ne .Project "") (not .PreservePs1) }}
$prevPrompt = $ExecutionContext.SessionState.PSVariable.GetValue('prompt')
if ($prevPrompt -eq $null) {
    $prevPrompt = "PS $PWD> "
}
function prompt {
    "[{{.Project}}] $prevPrompt"
}
{{end}}

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

echo "{{ escapePwsh .ActivatedMessage}}"

{{.UserScripts}}
