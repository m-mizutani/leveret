Generate a short title (less than {{.MaxLength}} characters) for this security alert. Return only the title without any explanation:

{{.AlertData}}
{{- if .FailedExamples}}

Previous attempts that were too long (please make it shorter):
{{- range .FailedExamples}}
- "{{.Title}}" ({{.Length}} characters)
{{- end}}
{{- end}}
