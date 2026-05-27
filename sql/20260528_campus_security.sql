USE lehu_video_db;

CREATE TABLE IF NOT EXISTS `campus_access_log` (
  `id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '游客为0',
  `ip` VARCHAR(64) NOT NULL DEFAULT '',
  `method` VARCHAR(12) NOT NULL DEFAULT '',
  `path` VARCHAR(255) NOT NULL DEFAULT '',
  `status_code` INT NOT NULL DEFAULT 0,
  `duration_ms` BIGINT NOT NULL DEFAULT 0,
  `user_agent` VARCHAR(512) NOT NULL DEFAULT '',
  `rate_limited` BOOLEAN NOT NULL DEFAULT FALSE,
  `blocked` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  INDEX `idx_campus_access_created` (`created_at`),
  INDEX `idx_campus_access_ip_created` (`ip`, `created_at`),
  INDEX `idx_campus_access_path_created` (`path`, `created_at`),
  INDEX `idx_campus_access_status_created` (`status_code`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园接口访问日志';

CREATE TABLE IF NOT EXISTS `campus_ip_block` (
  `id` BIGINT NOT NULL,
  `ip` VARCHAR(64) NOT NULL,
  `reason` VARCHAR(255) NOT NULL DEFAULT '',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1=生效 0=解除',
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_ip_block_ip` (`ip`),
  INDEX `idx_campus_ip_block_status` (`status`, `updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园接口 IP 封禁';
