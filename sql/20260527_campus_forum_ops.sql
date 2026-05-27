USE lehu_video_db;
SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `campus_forum_report` (
  `id` BIGINT NOT NULL,
  `target_type` VARCHAR(32) NOT NULL COMMENT 'post/comment',
  `target_id` BIGINT NOT NULL,
  `reporter_id` BIGINT NOT NULL,
  `reason` VARCHAR(64) NOT NULL DEFAULT '',
  `detail` VARCHAR(500) NOT NULL DEFAULT '',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '0=待处理 1=已处理 2=驳回',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_report_once` (`target_type`, `target_id`, `reporter_id`),
  INDEX `idx_campus_report_target` (`target_type`, `target_id`, `status`),
  INDEX `idx_campus_report_reporter` (`reporter_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园论坛举报记录';

