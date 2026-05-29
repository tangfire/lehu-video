USE lehu_campus_db;

CREATE TABLE IF NOT EXISTS `campus_notification` (
  `id` BIGINT NOT NULL,
  `recipient_id` BIGINT NOT NULL COMMENT '接收用户',
  `actor_id` BIGINT NOT NULL DEFAULT 0 COMMENT '触发用户，系统通知为运营用户或0',
  `event_type` VARCHAR(32) NOT NULL COMMENT 'comment/reply/post_like/post_collect/comment_like/system',
  `target_type` VARCHAR(32) NOT NULL DEFAULT '',
  `target_id` BIGINT NOT NULL DEFAULT 0,
  `dedupe_key` VARCHAR(191) DEFAULT NULL COMMENT '互动通知幂等键，系统通知为空',
  `title` VARCHAR(120) NOT NULL DEFAULT '',
  `content` VARCHAR(600) NOT NULL DEFAULT '',
  `link_page` VARCHAR(64) NOT NULL DEFAULT '',
  `link_params` JSON DEFAULT NULL,
  `read_at` DATETIME(3) DEFAULT NULL,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_notification_dedupe` (`dedupe_key`),
  INDEX `idx_campus_notification_user_created` (`recipient_id`, `is_deleted`, `created_at`, `id`),
  INDEX `idx_campus_notification_user_unread` (`recipient_id`, `read_at`, `is_deleted`, `created_at`),
  INDEX `idx_campus_notification_event` (`event_type`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园站内消息通知';

SET @db_name = DATABASE();

SET @sql = (
  SELECT IF(COUNT(*) = 0,
    'ALTER TABLE `campus_notification` ADD COLUMN `dedupe_key` VARCHAR(191) DEFAULT NULL COMMENT ''互动通知幂等键，系统通知为空'' AFTER `target_id`',
    'SELECT 1'
  )
  FROM information_schema.columns
  WHERE table_schema = @db_name
    AND table_name = 'campus_notification'
    AND column_name = 'dedupe_key'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `campus_notification`
SET `dedupe_key` = CONCAT(`recipient_id`, ':', `actor_id`, ':', `event_type`, ':', `target_type`, ':', `target_id`)
WHERE `dedupe_key` IS NULL
  AND `event_type` IN ('post_like', 'post_collect', 'comment_like')
  AND `recipient_id` > 0
  AND `actor_id` > 0
  AND `target_id` > 0;

SET @sql = (
  SELECT IF(COUNT(*) > 0,
    'ALTER TABLE `campus_notification` DROP INDEX `uk_campus_notification_interaction`',
    'SELECT 1'
  )
  FROM information_schema.statistics
  WHERE table_schema = @db_name
    AND table_name = 'campus_notification'
    AND index_name = 'uk_campus_notification_interaction'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(COUNT(*) = 0,
    'CREATE UNIQUE INDEX `uk_campus_notification_dedupe` ON `campus_notification` (`dedupe_key`)',
    'SELECT 1'
  )
  FROM information_schema.statistics
  WHERE table_schema = @db_name
    AND table_name = 'campus_notification'
    AND index_name = 'uk_campus_notification_dedupe'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
