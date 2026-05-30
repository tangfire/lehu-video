INSERT INTO `campus_ops_setting` (`setting_key`, `setting_value`, `updated_by`)
VALUES ('agent_audit_auto_pass_confidence', '0.85', 0)
ON DUPLICATE KEY UPDATE `setting_key` = `setting_key`;
