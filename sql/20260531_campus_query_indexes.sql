USE lehu_campus_db;

SET @sql := (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_forum_post` ADD INDEX `idx_campus_post_status_created` (`status`, `is_deleted`, `created_at`, `id`)',
    'SELECT 1'
  )
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'campus_forum_post'
    AND INDEX_NAME = 'idx_campus_post_status_created'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_ops_alert` ADD INDEX `idx_campus_ops_alert_status_sent` (`status`, `sent_at`, `id`)',
    'SELECT 1'
  )
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'campus_ops_alert'
    AND INDEX_NAME = 'idx_campus_ops_alert_status_sent'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_ops_alert` ADD INDEX `idx_campus_ops_alert_status_updated` (`status`, `updated_at`, `id`)',
    'SELECT 1'
  )
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'campus_ops_alert'
    AND INDEX_NAME = 'idx_campus_ops_alert_status_updated'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_ops_alert` ADD INDEX `idx_campus_ops_alert_feishu_updated` (`feishu_status`, `updated_at`, `id`)',
    'SELECT 1'
  )
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'campus_ops_alert'
    AND INDEX_NAME = 'idx_campus_ops_alert_feishu_updated'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_ops_alert` ADD INDEX `idx_campus_ops_alert_updated` (`updated_at`, `id`)',
    'SELECT 1'
  )
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'campus_ops_alert'
    AND INDEX_NAME = 'idx_campus_ops_alert_updated'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
