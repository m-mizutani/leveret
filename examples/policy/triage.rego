package triage

# デフォルトは通常アラートとして受理
default action = "accept"

default severity = "medium"

default note = ""

# 既知の誤検知パターンは棄却（最優先）
action = "discard" if {
	contains(input.alert.title, "scheduled maintenance")
}

severity = "info" if {
	contains(input.alert.title, "scheduled maintenance")
}

note = "Known false positive pattern" if {
	contains(input.alert.title, "scheduled maintenance")
}

# Enrich実行結果で悪意を検出した場合はcritical
severity = "critical" if {
	some exec in input.enrich
	contains(exec.result, "malicious")
}

note = "Malicious activity detected" if {
	some exec in input.enrich
	contains(exec.result, "malicious")
}

# Enrich実行結果で疑わしい活動を検出した場合はhigh
severity = "high" if {
	some exec in input.enrich
	contains(exec.result, "suspicious")
	not contains(exec.result, "malicious")
}

# 特定のenrich結果IDに基づく判定も可能
severity = "critical" if {
	some exec in input.enrich
	exec.id == "ip_threat_intel"
	contains(exec.result, "malicious")
}

# 元のアラートのseverityも参考にする
severity = "critical" if {
	some attr in input.alert.attributes
	attr.key == "severity"
	to_number(attr.value) >= 8
}

severity = "high" if {
	some attr in input.alert.attributes
	attr.key == "severity"
	sev := to_number(attr.value)
	sev >= 5
	sev < 8
}

severity = "low" if {
	some attr in input.alert.attributes
	attr.key == "severity"
	to_number(attr.value) < 2
}
