project: {{.Project}}
{{if and (.Private) (eq .CommitID "") }}
private: {{.Private}}
{{end}}
{{.Content}}