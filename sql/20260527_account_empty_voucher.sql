USE lehu_video_db;
SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

ALTER TABLE `account`
  MODIFY COLUMN `active_mobile` VARCHAR(20)
    GENERATED ALWAYS AS (IF(`is_deleted` = 0 AND `mobile` <> '', `mobile`, NULL)) STORED,
  MODIFY COLUMN `active_email` VARCHAR(100)
    GENERATED ALWAYS AS (IF(`is_deleted` = 0 AND `email` <> '', `email`, NULL)) STORED;
