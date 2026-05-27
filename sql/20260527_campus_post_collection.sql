USE `lehu_video_db`;

SET @column_exists := (
  SELECT COUNT(1)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'campus_forum_post'
    AND column_name = 'collected_count'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE `campus_forum_post` ADD COLUMN `collected_count` BIGINT NOT NULL DEFAULT 0 AFTER `comment_count`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS `campus_forum_post_collection` (
  `id` BIGINT NOT NULL,
  `post_id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_post_collection_user` (`post_id`, `user_id`),
  INDEX `idx_campus_post_collection_post` (`post_id`, `is_deleted`, `created_at`),
  INDEX `idx_campus_post_collection_user` (`user_id`, `is_deleted`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园社区笔记收藏';

UPDATE `campus_forum_post` p
LEFT JOIN (
  SELECT post_id, COUNT(*) AS collected_count
  FROM `campus_forum_post_collection`
  WHERE is_deleted = FALSE
  GROUP BY post_id
) c ON c.post_id = p.id
SET p.collected_count = COALESCE(c.collected_count, 0);
