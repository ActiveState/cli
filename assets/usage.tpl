Usage:
{{- if .Cobra.Runnable}}
    {{.Cobra.UseLine}}
{{- range  .Cmd.Arguments }} <{{ .Name }}>{{ end }}
{{- end}}
{{- if .Cobra.HasAvailableSubCommands}}
    {{.Cobra.CommandPath}} [command]
{{- end}}
{{- if gt (len .Cobra.Aliases) 0}}

Aliases:
    {{.Cobra.NameAndAliases}}
{{- end}}
{{- if .Cobra.HasExample}}

Examples:
    {{.Cobra.Example}}
{{- end}}

{{childCommands .Cmd}}
{{- if .Cobra.HasAvailableFlags}}

Flags:
{{.Cobra.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{- end}}
{{- if .Cobra.HasAvailableInheritedFlags}}

Global Flags:
{{.Cobra.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{- end}}
{{- if gt (len .Cmd.Arguments) 0}}

Arguments:
    {{-  range .Cmd.Arguments }}
  <{{ .Name }}> {{ if .Required }}          {{ else }}(optional){{ end }} {{ .Description }}
    {{-  end }}
{{- end}}
{{- if .Cobra.HasHelpSubCommands}}

Additional help topics:
{{- range .Cobra.Commands}}
    {{- if .Cobra.IsAdditionalHelpTopicCommand}}
    {{rpad .Cobra.CommandPath .Cobra.CommandPathPadding}} {{.Cobra.Short}}
    {{- end}}
{{- end}}
{{- end}}
{{- if .Cobra.HasAvailableSubCommands}}

Use "{{.Cobra.CommandPath}} [command] --help" for more information about a command.
{{- end}}