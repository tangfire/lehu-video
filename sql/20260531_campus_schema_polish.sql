USE lehu_campus_db;

SET @sql := (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_notification` ADD INDEX `idx_campus_notification_user_event_created` (`recipient_id`, `event_type`, `is_deleted`, `created_at`, `id`)',
    'SELECT 1'
  )
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'campus_notification'
    AND INDEX_NAME = 'idx_campus_notification_user_event_created'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_rag_eval_case` ADD COLUMN `source_log_key` BIGINT GENERATED ALWAYS AS (IF(`source_log_id` > 0, `source_log_id`, NULL)) STORED AFTER `source_log_id`',
    'SELECT 1'
  )
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'campus_rag_eval_case'
    AND COLUMN_NAME = 'source_log_key'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_rag_eval_case` ADD UNIQUE KEY `uk_campus_rag_eval_source_log` (`source_log_key`)',
    'SELECT 1'
  )
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'campus_rag_eval_case'
    AND INDEX_NAME = 'uk_campus_rag_eval_source_log'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
