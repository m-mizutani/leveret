package ingest

# SCC (Security Command Center) finding を受信してアラートに変換

alert contains {
	"title": sprintf("SCC: %s", [input.finding.category]),
	"description": input.finding.description,
	"attributes": extract_scc_attributes(input),
} if {
	input.finding
	input.finding.category
}

# SCC属性の抽出
extract_scc_attributes(input_data) = attrs if {
	attrs := array.concat(
		extract_basic_attributes(input_data.finding),
		extract_indicator_attributes(input_data.finding),
	)
}

# 基本属性
extract_basic_attributes(finding) = attrs if {
	attrs := [
		{
			"key": "severity",
			"value": finding.severity,
			"type": "string",
		},
		{
			"key": "finding_class",
			"value": finding.findingClass,
			"type": "string",
		},
		{
			"key": "category",
			"value": finding.category,
			"type": "string",
		},
		{
			"key": "resource_name",
			"value": finding.resourceName,
			"type": "string",
		},
		{
			"key": "project_id",
			"value": finding.sourceProperties.properties.instance_name,
			"type": "string",
		},
	]
}

# Indicator属性（IPアドレス、ドメインなど）
extract_indicator_attributes(finding) = attrs if {
	finding.indicator
	attrs := array.concat(
		extract_ip_addresses(finding.indicator),
		extract_domains(finding.indicator),
	)
}

extract_indicator_attributes(finding) = [] if {
	not finding.indicator
}

# IPアドレスの抽出
extract_ip_addresses(indicator) = attrs if {
	indicator.ipAddresses
	attrs := [attr |
		some ip in indicator.ipAddresses
		attr := {
			"key": "ip_address",
			"value": ip,
			"type": "ip_address",
		}
	]
}

extract_ip_addresses(indicator) = [] if {
	not indicator.ipAddresses
}

# ドメインの抽出
extract_domains(indicator) = attrs if {
	indicator.domains
	attrs := [attr |
		some domain in indicator.domains
		attr := {
			"key": "domain",
			"value": domain,
			"type": "string",
		}
	]
}

extract_domains(indicator) = [] if {
	not indicator.domains
}
