USE lehu_campus_db;

SET @db_name = DATABASE();

SET @sql = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_agent_run` ADD COLUMN `source` VARCHAR(24) NOT NULL DEFAULT ''manual'' COMMENT ''manual/scheduled'' AFTER `status`',
    'SELECT 1'
  )
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @db_name
    AND TABLE_NAME = 'campus_agent_run'
    AND COLUMN_NAME = 'source'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_agent_run` ADD COLUMN `feishu_sent_at` DATETIME(3) DEFAULT NULL AFTER `error_message`',
    'SELECT 1'
  )
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @db_name
    AND TABLE_NAME = 'campus_agent_run'
    AND COLUMN_NAME = 'feishu_sent_at'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_agent_run` ADD COLUMN `feishu_status` VARCHAR(24) NOT NULL DEFAULT ''pending'' COMMENT ''pending/sent/failed/skipped'' AFTER `feishu_sent_at`',
    'SELECT 1'
  )
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @db_name
    AND TABLE_NAME = 'campus_agent_run'
    AND COLUMN_NAME = 'feishu_status'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_agent_run` ADD COLUMN `feishu_error` VARCHAR(1000) NOT NULL DEFAULT '''' AFTER `feishu_status`',
    'SELECT 1'
  )
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = @db_name
    AND TABLE_NAME = 'campus_agent_run'
    AND COLUMN_NAME = 'feishu_error'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `campus_agent_run` ADD INDEX `idx_campus_agent_run_feishu` (`feishu_status`, `updated_at`)',
    'SELECT 1'
  )
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = @db_name
    AND TABLE_NAME = 'campus_agent_run'
    AND INDEX_NAME = 'idx_campus_agent_run_feishu'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
