{{if ne .Owner ""}}
#https://stackoverflow.com/questions/5725888/windows-powershell-changing-the-command-prompt
function Global:prompt {
    $currentDirectory = $(Get-Location)
    $UncRoot = $currentDirectory.Drive.DisplayRoot
    write-host " $UncRoot" -ForegroundColor Gray
    # Convert-Path needed for pure UNC-locations
    write-host "[{{.Owner}}/{{.Name}}] PS $(Convert-Path $currentDirectory)>" -NoNewline -ForegroundColor Yellow
    return " "
}
{{end}}

{{- range $K, $V := .Env}}
{{- if eq $K "PATH"}}
$env:{{$K}}="{{$V}}:$env:PATH"
{{- else}}
$env:{{$K}}="{{$V}}"
{{- end}}
{{- end}}

{{$execCmd := .ExecName}}

{{ if .ExecAlias }}
Set-Alias  {{.ExecName}} {{.ExecAlias}}
{{ end }}

{{range $K, $CMD := .Scripts}}
DOSKEY {{$K}}="{{$execCmd}}" run "{{$CMD}}" $*
{{end}}

{{range $K, $CMD := .Scripts}}
function {{$K}} {
    {{$.ExecName}} {{$CMD}} $args
}
export -f {{$K}}
{{end}}

cd {{.WD}}

{{range $line := splitLines .ActivatedMessage}}
    {{if eq $line ""}}write-host .{{else}}write-host {{$line}}{{end}}
{{end}}

{{.UserScripts}}
