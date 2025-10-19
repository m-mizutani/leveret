Generate a JSON response with a short title, detailed description, and extracted attributes for this security alert.

# Requirements:
- title: Must be less than {{.MaxTitleLength}} characters
- description: Should be 2-3 sentences explaining what happened and why it might be important
- attributes: Extract ONLY the most important attributes that are essential for investigation or analysis
  - Focus on IOCs (Indicators of Compromise): IP addresses, domains, hashes, URLs
  - Include key contextual information: affected usernames, critical resource names, error codes, unusual counts
  - DO NOT extract every field - be selective and include only what would help understand the threat or incident
  - Typically 3-10 attributes per alert is sufficient
  - Each attribute should have:
    - key: Descriptive name using snake_case (e.g., "source_ip", "user_name", "error_count", "api_endpoint")
    - value: The actual value as a string
    - type: Choose the most specific type:
      - "ip_address": For IPv4/IPv6 addresses
      - "domain": For domain names and hostnames
      - "hash": For file hashes (MD5, SHA1, SHA256, etc.)
      - "url": For URLs and URIs
      - "number": For numeric values (counts, sizes, IDs)
      - "string": For general text values (usernames, resource names, error messages, etc.)

# Alert data:
{{.AlertData}}
{{- if .FailedExamples}}

# Previous attempts failed with the following errors:
{{- range .FailedExamples}}
- {{.}}
{{- end}}
{{- end}}

Return the response as JSON with "title", "description", and "attributes" fields.
