package enrich

# IP属性がある場合は脅威調査
prompt contains {
	"id": "ip_threat_intel",
	"content": sprintf("Investigate IP address %s. Check if it's malicious using threat intelligence tools. Also check if it's in our internal network ranges.", [ip]),
	"format": "text",
} if {
	some attr in input.attributes
	attr.type == "ip_address"
	ip := attr.value
}

# 認証関連アラートはログ分析（JSON形式で構造化結果を要求）
prompt contains {
	"id": "auth_log_analysis",
	"content": sprintf("Alert: %s. Search BigQuery logs for related authentication failures in the last 24 hours. Return structured analysis.", [input.title]),
	"format": "json",
} if {
	contains(input.title, "authentication")
}

# 高Severityアラートは包括的調査
prompt contains {
	"id": "high_severity_investigation",
	"content": "This is a high severity alert. Perform comprehensive investigation using all available tools (threat intelligence, log analysis, etc.). Provide detailed findings.",
	"format": "text",
} if {
	some attr in input.attributes
	attr.key == "severity"
	to_number(attr.value) >= 7
}
