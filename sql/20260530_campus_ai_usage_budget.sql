USE lehu_campus_db;

CREATE TABLE IF NOT EXISTS `campus_ai_usage_log` (
  `id` BIGINT NOT NULL,
  `feature` VARCHAR(48) NOT NULL DEFAULT '' COMMENT 'content_audit/agent_copilot/ezai_reply/ezai_preview',
  `source_type` VARCHAR(48) NOT NULL DEFAULT '',
  `source_id` VARCHAR(64) NOT NULL DEFAULT '',
  `model` VARCHAR(64) NOT NULL DEFAULT '',
  `prompt_tokens` BIGINT NOT NULL DEFAULT 0,
  `completion_tokens` BIGINT NOT NULL DEFAULT 0,
  `total_tokens` BIGINT NOT NULL DEFAULT 0,
  `estimated_cost_usd` DECIMAL(12,8) NOT NULL DEFAULT 0,
  `estimated_cost_cny` DECIMAL(12,6) NOT NULL DEFAULT 0,
  `status` VARCHAR(24) NOT NULL DEFAULT 'success' COMMENT 'success/failed/skipped',
  `error_message` VARCHAR(1000) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  INDEX `idx_campus_ai_usage_created` (`created_at`),
  INDEX `idx_campus_ai_usage_feature_created` (`feature`, `created_at`),
  INDEX `idx_campus_ai_usage_source` (`source_type`, `source_id`),
  INDEX `idx_campus_ai_usage_status` (`status`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园AI模型调用与成本账本';

INSERT INTO `campus_ops_setting` (`setting_key`, `setting_value`, `updated_by`) VALUES
('ai_budget_enabled', 'true', 0),
('ai_monthly_budget_cny', '5', 0),
('ai_daily_budget_cny', '0.5', 0),
('ai_budget_warn_ratio', '0.7,0.9', 0)
ON DUPLICATE KEY UPDATE `setting_key` = `setting_key`;
