CREATE TABLE IF NOT EXISTS `campus_ops_setting` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `setting_key` VARCHAR(64) NOT NULL,
  `setting_value` VARCHAR(512) NOT NULL DEFAULT '',
  `updated_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_ops_setting_key` (`setting_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园运营配置';

INSERT INTO `campus_ops_setting` (`setting_key`, `setting_value`, `updated_by`)
VALUES ('post_audit_mode', 'off', 0)
ON DUPLICATE KEY UPDATE `setting_key` = `setting_key`;

CREATE TABLE IF NOT EXISTS `campus_ai_audit_task` (
  `id` BIGINT NOT NULL,
  `target_type` VARCHAR(32) NOT NULL DEFAULT 'post',
  `target_id` BIGINT NOT NULL,
  `status` VARCHAR(24) NOT NULL DEFAULT 'pending' COMMENT 'pending/processing/done/failed',
  `risk_level` VARCHAR(24) NOT NULL DEFAULT '',
  `decision` VARCHAR(24) NOT NULL DEFAULT '',
  `reason` VARCHAR(255) NOT NULL DEFAULT '',
  `raw_result` TEXT,
  `retry_count` INT NOT NULL DEFAULT 0,
  `next_retry_at` DATETIME(3) DEFAULT NULL,
  `locked_until` DATETIME(3) DEFAULT NULL,
  `last_error` VARCHAR(600) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `processed_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_ai_audit_target` (`target_type`, `target_id`),
  INDEX `idx_campus_ai_audit_status_next` (`status`, `next_retry_at`, `locked_until`, `id`),
  INDEX `idx_campus_ai_audit_target_created` (`target_type`, `target_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园AI内容审核任务';
