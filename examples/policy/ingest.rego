package ingest

# AWS GuardDutyアラートの属性抽出
alert contains {
	"title": sprintf("GuardDuty: %s", [input.type]),
	"description": sprintf("Severity %d alert in %s", [input.severity, input.region]),
	"attributes": extract_guardduty_attributes(input),
} if {
	input.service.serviceName == "guardduty"
	# 棄却条件: テストアラートやホワイトリストIPは除外
	not is_test_alert
	not is_whitelisted
}

# ヘルパー: テストアラート判定
is_test_alert if {
	input.environment == "development"
	input.test == true
}

# ヘルパー: ホワイトリスト判定
is_whitelisted if {
	some ip in input.source_ips
	ip in data.whitelist.ips
}

# GuardDutyの属性抽出ヘルパー
extract_guardduty_attributes(alert_data) = attrs if {
	attrs := [
		{
			"key": "aws_account_id",
			"value": alert_data.accountId,
			"type": "string",
		},
		{
			"key": "resource_type",
			"value": alert_data.resource.resourceType,
			"type": "string",
		},
		{
			"key": "severity",
			"value": sprintf("%d", [alert_data.severity]),
			"type": "number",
		},
		{
			"key": "region",
			"value": alert_data.region,
			"type": "string",
		},
	]
}
