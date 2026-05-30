INSERT INTO `campus_ops_setting` (`setting_key`, `setting_value`, `is_public`)
VALUES
('ai_monthly_budget_cny', '5', 0),
('ai_daily_budget_cny', '0.5', 0)
ON DUPLICATE KEY UPDATE `setting_key` = `setting_key`;

UPDATE `campus_ops_setting`
SET `setting_value` = '5'
WHERE `setting_key` = 'ai_monthly_budget_cny'
  AND `setting_value` IN ('20', '20.0', '20.00');

UPDATE `campus_ops_setting`
SET `setting_value` = '0.5'
WHERE `setting_key` = 'ai_daily_budget_cny'
  AND `setting_value` IN ('2', '2.0', '2.00');
