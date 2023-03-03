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

{{- if .Cmd.Examples  }}

Examples:
{{- range  .Cmd.Examples }}
    {{ . -}}
{{- end }}
{{- end }}

{{- childCommands .Cmd}}
{{- if gt (len .Cmd.Arguments) 0}}

Arguments:
{{-  range .Cmd.Arguments }}
  <{{ .Name }}> {{ if .Required }}          {{ else }}(optional){{ end }} {{ .Description }}
{{-  end }}
{{-  end }}

{{- if .Cobra.HasAvailableFlags}}

Flags:
{{.Cobra.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{- end}}
{{- if .Cobra.HasAvailableInheritedFlags}}

Global Flags:
{{.Cobra.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
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

{{- if .OptinUnstable }}

WARNING: You have an access to list of full commands, including unstable features still in beta, in order to hide these features run:

"state config set optin.unstable false"
{{- else }}

To access the list of full commands, including unstable features still in beta, run:

"state config set optin.unstable true"
{{- end}}
{{- end}}
