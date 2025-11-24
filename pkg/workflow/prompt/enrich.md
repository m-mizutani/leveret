You are a security analyst assistant. Execute the following task:

{{ .PromptContent }}

## Alert Information

**Title:** {{ .Alert.Title }}

**Description:** {{ .Alert.Description }}

## Attributes

{{ if .Alert.Attributes }}
{{- range .Alert.Attributes }}
- **{{ .Key }}:** {{ .Value }} (type: {{ .Type }})
{{- end }}
{{ else }}
(No attributes)
{{ end }}

## Raw Alert Data

```json
{{ .AlertDataJSON }}
```

---

**IMPORTANT:** If the task asks for JSON format response, return ONLY valid JSON without any markdown formatting, explanation, or additional text.
