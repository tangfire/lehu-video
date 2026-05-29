USE lehu_campus_db;

CREATE TABLE IF NOT EXISTS `campus_feedback` (
  `id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `feedback_type` VARCHAR(32) NOT NULL DEFAULT 'suggestion' COMMENT 'bug/suggestion/content/cooperation/contact',
  `content` VARCHAR(1000) NOT NULL,
  `contact` VARCHAR(120) NOT NULL DEFAULT '',
  `images` JSON DEFAULT NULL,
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '0=待处理 1=处理中 2=已处理',
  `operator_note` VARCHAR(500) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  INDEX `idx_campus_feedback_status_created` (`status`, `created_at`),
  INDEX `idx_campus_feedback_user_created` (`user_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园小程序用户反馈';
