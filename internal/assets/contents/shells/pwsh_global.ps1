{{if and (ne .Project "") (not .PreservePs1) }}
$prevPrompt = $ExecutionContext.SessionState.PSVariable.GetValue('prompt')
if ($prevPrompt -eq $null) {
    $prevPrompt = "PS $PWD> "
}
function prompt {
    "[{{.Project}}] $prevPrompt"
}
{{end}}

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
$env:{{$K}} = "{{ escapePwsh $V }};$env:PATH"
{{- else}}
$env:{{$K}} = "{{ escapePwsh $V }}"
{{- end}}
{{- end}}