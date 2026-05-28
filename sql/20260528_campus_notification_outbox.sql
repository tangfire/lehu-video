USE lehu_video_db;

CREATE TABLE IF NOT EXISTS `campus_notification_outbox` (
  `id` BIGINT NOT NULL,
  `recipient_id` BIGINT NOT NULL DEFAULT 0 COMMENT '互动通知接收用户，系统群发为0',
  `actor_id` BIGINT NOT NULL DEFAULT 0 COMMENT '触发用户或运营用户',
  `event_type` VARCHAR(32) NOT NULL COMMENT 'comment/reply/post_like/post_collect/comment_like/system',
  `target_type` VARCHAR(32) NOT NULL DEFAULT '',
  `target_id` BIGINT NOT NULL DEFAULT 0,
  `dedupe_key` VARCHAR(191) DEFAULT NULL COMMENT '投递幂等键',
  `title` VARCHAR(120) NOT NULL DEFAULT '',
  `content` VARCHAR(600) NOT NULL DEFAULT '',
  `link_page` VARCHAR(64) NOT NULL DEFAULT '',
  `link_params` JSON DEFAULT NULL,
  `audience` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '系统通知范围，v1=all_users',
  `status` VARCHAR(24) NOT NULL DEFAULT 'pending' COMMENT 'pending/processing/done/failed',
  `retry_count` INT NOT NULL DEFAULT 0,
  `next_retry_at` DATETIME(3) DEFAULT NULL,
  `locked_until` DATETIME(3) DEFAULT NULL,
  `last_error` VARCHAR(600) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `processed_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_notification_outbox_dedupe` (`dedupe_key`),
  INDEX `idx_campus_notification_outbox_status_next` (`status`, `next_retry_at`, `locked_until`, `id`),
  INDEX `idx_campus_notification_outbox_created` (`created_at`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园通知可靠投递任务';
