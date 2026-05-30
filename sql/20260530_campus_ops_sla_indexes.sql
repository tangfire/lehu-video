USE lehu_campus_db;

SET @sql := (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_forum_report` ADD INDEX `idx_campus_report_status_created` (`status`, `created_at`)',
    'SELECT 1'
  )
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'campus_forum_report'
    AND INDEX_NAME = 'idx_campus_report_status_created'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_forum_comment` ADD INDEX `idx_campus_comment_status_created` (`status`, `is_deleted`, `created_at`)',
    'SELECT 1'
  )
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'campus_forum_comment'
    AND INDEX_NAME = 'idx_campus_comment_status_created'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
