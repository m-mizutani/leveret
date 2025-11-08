-- title: Admin Activities
-- description: Query to track administrative activities and configuration changes

SELECT
  timestamp,
  protopayload_auditlog.authenticationInfo.principalEmail as principal,
  protopayload_auditlog.resourceName as resource,
  protopayload_auditlog.methodName as method,
  protopayload_auditlog.serviceName as service
FROM
  `mztn-audit.google_cloud_audit.cloudaudit_googleapis_com_activity`
WHERE
  TIMESTAMP_TRUNC(timestamp, DAY) >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
  AND (
    protopayload_auditlog.methodName LIKE '%create%'
    OR protopayload_auditlog.methodName LIKE '%delete%'
    OR protopayload_auditlog.methodName LIKE '%update%'
    OR protopayload_auditlog.methodName LIKE '%setIamPolicy%'
  )
ORDER BY
  timestamp DESC
LIMIT 100
