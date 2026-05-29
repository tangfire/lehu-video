USE lehu_campus_db;

SET @db_name = DATABASE();

SET @sql = (
  SELECT IF(COUNT(*) = 0,
    'ALTER TABLE `campus_forum_post` ADD COLUMN `is_official` BOOLEAN NOT NULL DEFAULT FALSE COMMENT ''官方/运营内容'' AFTER `cover_url`',
    'SELECT 1'
  )
  FROM information_schema.columns
  WHERE table_schema = @db_name
    AND table_name = 'campus_forum_post'
    AND column_name = 'is_official'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(COUNT(*) = 0,
    'ALTER TABLE `campus_forum_post` ADD COLUMN `is_featured` BOOLEAN NOT NULL DEFAULT FALSE COMMENT ''精选推荐'' AFTER `is_official`',
    'SELECT 1'
  )
  FROM information_schema.columns
  WHERE table_schema = @db_name
    AND table_name = 'campus_forum_post'
    AND column_name = 'is_featured'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(COUNT(*) = 0,
    'ALTER TABLE `campus_forum_post` ADD COLUMN `sort_weight` INT NOT NULL DEFAULT 0 COMMENT ''运营排序权重'' AFTER `is_featured`',
    'SELECT 1'
  )
  FROM information_schema.columns
  WHERE table_schema = @db_name
    AND table_name = 'campus_forum_post'
    AND column_name = 'sort_weight'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(COUNT(*) = 0,
    'ALTER TABLE `campus_forum_post` ADD COLUMN `is_pinned` BOOLEAN NOT NULL DEFAULT FALSE COMMENT ''首页置顶'' AFTER `is_featured`',
    'SELECT 1'
  )
  FROM information_schema.columns
  WHERE table_schema = @db_name
    AND table_name = 'campus_forum_post'
    AND column_name = 'is_pinned'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(COUNT(*) > 0,
    'ALTER TABLE `campus_forum_post` DROP INDEX `idx_campus_post_ops_sort`',
    'SELECT 1'
  )
  FROM information_schema.statistics
  WHERE table_schema = @db_name
    AND table_name = 'campus_forum_post'
    AND index_name = 'idx_campus_post_ops_sort'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(COUNT(*) = 0,
    'CREATE INDEX `idx_campus_post_ops_sort` ON `campus_forum_post` (`status`, `is_deleted`, `is_pinned`, `is_featured`, `sort_weight`, `created_at`)',
    'SELECT 1'
  )
  FROM information_schema.statistics
  WHERE table_schema = @db_name
    AND table_name = 'campus_forum_post'
    AND index_name = 'idx_campus_post_ops_sort'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS `campus_operator` (
  `user_id` BIGINT NOT NULL,
  `role` VARCHAR(24) NOT NULL DEFAULT 'operator' COMMENT 'operator/admin',
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`user_id`),
  INDEX `idx_campus_operator_role` (`role`, `is_deleted`, `updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园运营后台权限';
