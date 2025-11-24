package triage

# デフォルトは受理
default action = "accept"

default severity = "high"

default note = ""

# Enrich結果がfalse_positiveの場合は棄却
action = "discard" if {
	some exec in input.enrich
	exec.id == "bigquery_impact_analysis"
	analysis := json.unmarshal(exec.result)
	analysis.result == "false_positive"
}

severity = "info" if {
	some exec in input.enrich
	exec.id == "bigquery_impact_analysis"
	analysis := json.unmarshal(exec.result)
	analysis.result == "false_positive"
}

note = reasoning if {
	some exec in input.enrich
	exec.id == "bigquery_impact_analysis"
	analysis := json.unmarshal(exec.result)
	analysis.result == "false_positive"
	reasoning := sprintf("False positive: %s", [analysis.reasoning])
}

# 確認された脅威の場合はcritical
severity = "critical" if {
	some exec in input.enrich
	exec.id == "bigquery_impact_analysis"
	analysis := json.unmarshal(exec.result)
	analysis.result == "confirmed"
}

note = reasoning if {
	some exec in input.enrich
	exec.id == "bigquery_impact_analysis"
	analysis := json.unmarshal(exec.result)
	analysis.result == "confirmed"
	reasoning := sprintf("Confirmed threat: %s", [analysis.reasoning])
}

# 元のSCC severityも考慮
severity = "critical" if {
	some attr in input.alert.attributes
	attr.key == "severity"
	attr.value == "CRITICAL"
	# Enrich結果がない、またはfalse_positiveでない場合
	not is_false_positive
}

severity = "high" if {
	some attr in input.alert.attributes
	attr.key == "severity"
	attr.value == "HIGH"
	not is_false_positive
}

severity = "medium" if {
	some attr in input.alert.attributes
	attr.key == "severity"
	attr.value == "MEDIUM"
	not is_false_positive
}

# false_positive判定ヘルパー
is_false_positive if {
	some exec in input.enrich
	exec.id == "bigquery_impact_analysis"
	analysis := json.unmarshal(exec.result)
	analysis.result == "false_positive"
}
