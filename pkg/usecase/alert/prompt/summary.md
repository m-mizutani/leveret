Generate a JSON response with a short title and detailed description for this security alert.

# Requirements:
- title: Must be less than {{.MaxTitleLength}} characters
- description: Should be 2-3 sentences explaining what happened and why it might be important

# Alert data:
{{.AlertData}}
{{- if .FailedExamples}}

# Previous attempts failed with the following errors:
{{- range .FailedExamples}}
- {{.}}
{{- end}}
{{- end}}

Return the response as JSON with "title" and "description" fields.
