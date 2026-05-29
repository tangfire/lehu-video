USE lehu_campus_db;

SET @schema_name = DATABASE();

SET @sql = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_forum_comment` ADD COLUMN `parent_id` BIGINT NOT NULL DEFAULT 0 COMMENT ''一级评论 ID，0 表示根评论''',
    'SELECT 1'
  )
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = @schema_name AND TABLE_NAME = 'campus_forum_comment' AND COLUMN_NAME = 'parent_id'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_forum_comment` ADD COLUMN `reply_to_comment_id` BIGINT NOT NULL DEFAULT 0 COMMENT ''回复的评论 ID''',
    'SELECT 1'
  )
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = @schema_name AND TABLE_NAME = 'campus_forum_comment' AND COLUMN_NAME = 'reply_to_comment_id'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_forum_comment` ADD COLUMN `reply_to_user_id` BIGINT NOT NULL DEFAULT 0 COMMENT ''回复的用户 ID''',
    'SELECT 1'
  )
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = @schema_name AND TABLE_NAME = 'campus_forum_comment' AND COLUMN_NAME = 'reply_to_user_id'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_forum_comment` ADD COLUMN `reply_count` BIGINT NOT NULL DEFAULT 0',
    'SELECT 1'
  )
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = @schema_name AND TABLE_NAME = 'campus_forum_comment' AND COLUMN_NAME = 'reply_count'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_forum_comment` ADD INDEX `idx_campus_comment_parent_created` (`parent_id`, `status`, `is_deleted`, `created_at`, `id`)',
    'SELECT 1'
  )
  FROM INFORMATION_SCHEMA.STATISTICS
  WHERE TABLE_SCHEMA = @schema_name AND TABLE_NAME = 'campus_forum_comment' AND INDEX_NAME = 'idx_campus_comment_parent_created'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS `campus_forum_comment_like` (
  `id` BIGINT NOT NULL,
  `comment_id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_comment_like_user` (`comment_id`, `user_id`),
  INDEX `idx_campus_comment_like_comment` (`comment_id`, `is_deleted`, `created_at`),
  INDEX `idx_campus_comment_like_user` (`user_id`, `is_deleted`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园论坛评论点赞';
