CREATE TABLE IF NOT EXISTS `campus_agent_run` (
  `id` BIGINT NOT NULL,
  `run_type` VARCHAR(32) NOT NULL DEFAULT '',
  `question` VARCHAR(1000) NOT NULL DEFAULT '',
  `status` VARCHAR(24) NOT NULL DEFAULT 'running' COMMENT 'running/done/failed',
  `summary` VARCHAR(500) NOT NULL DEFAULT '',
  `risk_level` VARCHAR(16) NOT NULL DEFAULT 'low',
  `result_json` JSON DEFAULT NULL,
  `tool_trace_json` JSON DEFAULT NULL,
  `error_message` VARCHAR(1000) NOT NULL DEFAULT '',
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  INDEX `idx_campus_agent_run_type` (`run_type`, `created_at`),
  INDEX `idx_campus_agent_run_status` (`status`, `created_at`),
  INDEX `idx_campus_agent_run_creator` (`created_by`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运营Copilot Agent运行记录';
