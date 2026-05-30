USE lehu_campus_db;

CREATE TABLE IF NOT EXISTS `campus_ops_alert` (
  `id` BIGINT NOT NULL,
  `alert_type` VARCHAR(48) NOT NULL DEFAULT '',
  `priority` VARCHAR(24) NOT NULL DEFAULT 'normal' COMMENT 'low/normal/high/critical',
  `target_type` VARCHAR(32) NOT NULL DEFAULT '',
  `target_id` BIGINT NOT NULL DEFAULT 0,
  `dedupe_key` VARCHAR(160) NOT NULL DEFAULT '',
  `title` VARCHAR(160) NOT NULL DEFAULT '',
  `summary` VARCHAR(800) NOT NULL DEFAULT '',
  `payload_json` JSON DEFAULT NULL,
  `status` VARCHAR(24) NOT NULL DEFAULT 'pending' COMMENT 'pending/processing/sent/skipped/failed',
  `feishu_status` VARCHAR(24) NOT NULL DEFAULT 'pending',
  `feishu_error` VARCHAR(1000) NOT NULL DEFAULT '',
  `retry_count` INT NOT NULL DEFAULT 0,
  `next_retry_at` DATETIME(3) DEFAULT NULL,
  `locked_until` DATETIME(3) DEFAULT NULL,
  `sent_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_ops_alert_dedupe` (`dedupe_key`),
  INDEX `idx_campus_ops_alert_status_next` (`status`, `next_retry_at`, `locked_until`, `id`),
  INDEX `idx_campus_ops_alert_target` (`target_type`, `target_id`),
  INDEX `idx_campus_ops_alert_type_created` (`alert_type`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运营值班Agent飞书提醒队列';

CREATE TABLE IF NOT EXISTS `campus_ops_action_token` (
  `id` BIGINT NOT NULL,
  `token_hash` CHAR(64) NOT NULL,
  `action` VARCHAR(32) NOT NULL DEFAULT '',
  `target_type` VARCHAR(32) NOT NULL DEFAULT '',
  `target_id` BIGINT NOT NULL DEFAULT 0,
  `reason` VARCHAR(255) NOT NULL DEFAULT '',
  `status` VARCHAR(24) NOT NULL DEFAULT 'active' COMMENT 'active/used/expired',
  `expires_at` DATETIME(3) NOT NULL,
  `used_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_ops_action_token_hash` (`token_hash`),
  INDEX `idx_campus_ops_action_token_target` (`target_type`, `target_id`),
  INDEX `idx_campus_ops_action_token_status_expire` (`status`, `expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='飞书审核按钮一次性动作Token';
