USE `lehu_campus_db`;

SET @column_exists := (
  SELECT COUNT(1)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'campus_forum_post'
    AND column_name = 'post_type'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE `campus_forum_post` ADD COLUMN `post_type` VARCHAR(24) NOT NULL DEFAULT ''note'' COMMENT ''note/lost/question/guide/club'' AFTER `media_type`',
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
    AND column_name = 'extra'
);
SET @ddl := IF(
  @column_exists = 0,
  'ALTER TABLE `campus_forum_post` ADD COLUMN `extra` JSON DEFAULT NULL AFTER `post_type`',
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
    AND index_name = 'idx_campus_post_type'
);
SET @ddl := IF(
  @idx_exists = 0,
  'CREATE INDEX `idx_campus_post_type` ON `campus_forum_post` (`post_type`, `status`, `is_deleted`, `created_at`)',
  'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `campus_forum_post`
SET `post_type` = 'note'
WHERE `post_type` IS NULL OR `post_type` = '';

UPDATE `campus_forum_post`
SET `extra` = JSON_OBJECT()
WHERE `extra` IS NULL;

INSERT INTO `campus_forum_category` (`id`, `code`, `name`, `description`, `sort_order`)
VALUES
  (1001, 'study', '学习交流', '课程讨论、资料分享、学习互助', 10),
  (1002, 'life', '生活求助', '失物招领、校园攻略、生活问题', 20),
  (1003, 'club', '社团活动', '招新、活动发布、组队约伴', 30),
  (1004, 'lost', '失物招领', '丢失、捡到、认领信息', 40),
  (1005, 'qa', '问答互助', '新生提问、同学答疑、校园经验', 50),
  (1006, 'guide', '校园攻略', '报到、宿舍、交通、生活指南', 60)
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `description` = VALUES(`description`),
  `sort_order` = VALUES(`sort_order`),
  `is_deleted` = FALSE,
  `updated_at` = CURRENT_TIMESTAMP(3);
