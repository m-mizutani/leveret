-- title: Failed Operations
-- description: Query to find operations that resulted in errors or failures

SELECT
  timestamp,
  protopayload_auditlog.authenticationInfo.principalEmail as principal,
  protopayload_auditlog.resourceName as resource,
  protopayload_auditlog.methodName as method,
  protopayload_auditlog.status.code as status_code,
  protopayload_auditlog.status.message as error_message
FROM
  `mztn-audit.google_cloud_audit.cloudaudit_googleapis_com_activity`
WHERE
  timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
  AND protopayload_auditlog.status.code IS NOT NULL
  AND protopayload_auditlog.status.code != 0
ORDER BY
  timestamp DESC
LIMIT 100
