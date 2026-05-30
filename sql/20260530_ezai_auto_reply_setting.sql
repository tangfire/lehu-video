USE lehu_campus_db;

INSERT INTO `campus_ops_setting` (`setting_key`, `setting_value`, `updated_by`) VALUES
('ezai_auto_reply_enabled', 'true', 0)
ON DUPLICATE KEY UPDATE `setting_key` = `setting_key`;
