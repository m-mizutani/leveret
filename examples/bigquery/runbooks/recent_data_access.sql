-- title: Recent Data Access Logs
-- description: Query to retrieve recent data access audit logs from the last 24 hours

SELECT
  timestamp,
  protopayload_auditlog.authenticationInfo.principalEmail as principal,
  protopayload_auditlog.resourceName as resource,
  protopayload_auditlog.methodName as method,
  protopayload_auditlog.serviceName as service
FROM
  `mztn-audit.google_cloud_audit.cloudaudit_googleapis_com_data_access`
WHERE
  timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)
ORDER BY
  timestamp DESC
LIMIT 100
