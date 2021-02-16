{{$currentGroup := ""}}
{{- range .Commands}}
{{- if and (ne $currentGroup .Group.String) (ne .Group.String "")}}
{{- $currentGroup = .Group.String}}

## {{.Group.String}}{{end}}

### {{.NameRecursive}}
{{.Description}}

**Usage**
```text
{{.NameRecursive}} {{if .Flags}}[flags]{{end}}{{range .Arguments}} <{{ .Name }}>{{end}}
```

{{- if .Arguments}}

**Arguments**
{{-  range .Arguments }}
* `<{{ .Name }}>`{{ if not .Required }} (optional){{ end }} {{ .Description }}
{{-  end }}
{{- end}}
{{- if .Flags}}

**Flags**
{{- range .Flags}}
* `--{{.Name}}`{{if .Shorthand}}, `-{{.Shorthand}}`{{end}} {{.Description}}
{{- end}}
{{- end}}
{{- end}}