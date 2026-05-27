USE `lehu_video_db`;

SET @column_exists := (
  SELECT COUNT(1)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'campus_forum_post'
    AND column_name = 'media_type'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE `campus_forum_post` ADD COLUMN `media_type` VARCHAR(16) NOT NULL DEFAULT ''text'' COMMENT ''text/image/video'' AFTER `images`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists := (
  SELECT COUNT(1)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'campus_forum_post'
    AND column_name = 'cover_url'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE `campus_forum_post` ADD COLUMN `cover_url` VARCHAR(1024) NOT NULL DEFAULT '''' AFTER `media_type`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @column_exists := (
  SELECT COUNT(1)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'campus_forum_post'
    AND column_name = 'video_url'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE `campus_forum_post` ADD COLUMN `video_url` VARCHAR(1024) NOT NULL DEFAULT '''' AFTER `cover_url`',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @idx_exists := (
  SELECT COUNT(1)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'campus_forum_post'
    AND index_name = 'idx_campus_post_media'
);
SET @ddl := IF(
  @idx_exists = 0,
  'CREATE INDEX `idx_campus_post_media` ON `campus_forum_post` (`media_type`, `status`, `is_deleted`, `created_at`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `campus_forum_post`
SET
  `media_type` = 'image',
  `cover_url` = COALESCE(JSON_UNQUOTE(JSON_EXTRACT(`images`, '$[0]')), '')
WHERE JSON_LENGTH(`images`) > 0
  AND (`media_type` = '' OR `media_type` = 'text');

UPDATE `campus_forum_post`
SET `media_type` = 'text'
WHERE `media_type` IS NULL OR `media_type` = '';
