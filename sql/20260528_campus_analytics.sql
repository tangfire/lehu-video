USE lehu_campus_db;

CREATE TABLE IF NOT EXISTS `campus_event` (
  `id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '游客为0',
  `event_type` VARCHAR(32) NOT NULL COMMENT 'visit/share/login/post_create/comment_create/like/collect',
  `page` VARCHAR(64) NOT NULL DEFAULT '',
  `target_type` VARCHAR(32) NOT NULL DEFAULT '',
  `target_id` BIGINT NOT NULL DEFAULT 0,
  `channel` VARCHAR(64) NOT NULL DEFAULT '',
  `extra` JSON DEFAULT NULL,
  `user_agent` VARCHAR(512) NOT NULL DEFAULT '',
  `ip` VARCHAR(64) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  INDEX `idx_campus_event_type_created` (`event_type`, `created_at`),
  INDEX `idx_campus_event_user_created` (`user_id`, `created_at`),
  INDEX `idx_campus_event_target_created` (`target_type`, `target_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园小程序轻量行为埋点';
