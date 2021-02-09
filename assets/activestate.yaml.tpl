project: {{.Project}}
{{if .IncludeLanguageField }}
languages: # Please run 'state push' to create your language runtime, once you do the language entry here will be removed
  - name: {{.LanguageName}}
    version: {{.LanguageVersion}}
private: {{.Private}}
{{end}}
{{.Content}}