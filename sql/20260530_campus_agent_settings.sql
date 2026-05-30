USE lehu_campus_db;

INSERT INTO `campus_ops_setting` (`setting_key`, `setting_value`, `updated_by`) VALUES
('agent_enabled', 'true', 0),
('agent_audit_enabled', 'true', 0),
('agent_audit_auto_pass_confidence', '0.85', 0),
('feishu_ops_enabled', 'true', 0),
('daily_report_enabled', 'true', 0),
('high_risk_notify_enabled', 'true', 0),
('report_notify_enabled', 'true', 0),
('feedback_notify_enabled', 'true', 0)
ON DUPLICATE KEY UPDATE `setting_key` = `setting_key`;
