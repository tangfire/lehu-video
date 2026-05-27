USE lehu_video_db;
SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `campus_wechat_identity` (
  `id` BIGINT NOT NULL,
  `provider` VARCHAR(32) NOT NULL DEFAULT 'wechat',
  `open_id` VARCHAR(128) NOT NULL,
  `union_id` VARCHAR(128) DEFAULT NULL,
  `user_id` BIGINT NOT NULL,
  `account_id` BIGINT NOT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_wechat_provider_openid` (`provider`, `open_id`),
  INDEX `idx_campus_wechat_user` (`user_id`),
  INDEX `idx_campus_wechat_account` (`account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园小程序微信身份绑定';

CREATE TABLE IF NOT EXISTS `campus_profile` (
  `id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `account_id` BIGINT NOT NULL,
  `open_id` VARCHAR(128) NOT NULL,
  `union_id` VARCHAR(128) DEFAULT NULL,
  `school_name` VARCHAR(100) NOT NULL DEFAULT '深圳职业技术大学深汕校区',
  `student_no` VARCHAR(64) DEFAULT NULL,
  `real_name` VARCHAR(64) DEFAULT NULL,
  `class_name` VARCHAR(100) DEFAULT NULL,
  `dorm_building` VARCHAR(64) DEFAULT NULL,
  `room_no` VARCHAR(64) DEFAULT NULL,
  `mobile` VARCHAR(20) DEFAULT NULL,
  `auth_status` TINYINT NOT NULL DEFAULT 0 COMMENT '0=未认证 1=已认证',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_profile_user` (`user_id`),
  UNIQUE KEY `uk_campus_profile_openid` (`open_id`),
  INDEX `idx_campus_profile_student` (`student_no`),
  INDEX `idx_campus_profile_auth` (`auth_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园身份资料';

CREATE TABLE IF NOT EXISTS `campus_forum_category` (
  `id` BIGINT NOT NULL,
  `code` VARCHAR(32) NOT NULL,
  `name` VARCHAR(32) NOT NULL,
  `description` VARCHAR(255) NOT NULL DEFAULT '',
  `sort_order` INT NOT NULL DEFAULT 0,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_forum_category_code` (`code`),
  INDEX `idx_campus_forum_category_sort` (`is_deleted`, `sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园论坛版块';

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

CREATE TABLE IF NOT EXISTS `campus_forum_post` (
  `id` BIGINT NOT NULL,
  `category_code` VARCHAR(32) NOT NULL,
  `author_id` BIGINT NOT NULL,
  `title` VARCHAR(120) NOT NULL,
  `content` TEXT NOT NULL,
  `images` JSON DEFAULT NULL,
  `media_type` VARCHAR(16) NOT NULL DEFAULT 'text' COMMENT 'text/image/video',
  `post_type` VARCHAR(24) NOT NULL DEFAULT 'note' COMMENT 'note/lost/question/guide/club',
  `extra` JSON DEFAULT NULL,
  `cover_url` VARCHAR(1024) NOT NULL DEFAULT '',
  `video_url` VARCHAR(1024) NOT NULL DEFAULT '',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '0=待审核 1=可见 2=拒绝 3=删除',
  `audit_reason` VARCHAR(255) NOT NULL DEFAULT '',
  `like_count` BIGINT NOT NULL DEFAULT 0,
  `comment_count` BIGINT NOT NULL DEFAULT 0,
  `collected_count` BIGINT NOT NULL DEFAULT 0,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  INDEX `idx_campus_post_category_created` (`category_code`, `status`, `is_deleted`, `created_at`, `id`),
  INDEX `idx_campus_post_author` (`author_id`, `is_deleted`, `created_at`),
  INDEX `idx_campus_post_hot` (`status`, `is_deleted`, `like_count`, `comment_count`, `created_at`),
  INDEX `idx_campus_post_media` (`media_type`, `status`, `is_deleted`, `created_at`),
  INDEX `idx_campus_post_type` (`post_type`, `status`, `is_deleted`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园论坛帖子';

CREATE TABLE IF NOT EXISTS `campus_forum_comment` (
  `id` BIGINT NOT NULL,
  `post_id` BIGINT NOT NULL,
  `author_id` BIGINT NOT NULL,
  `content` VARCHAR(1000) NOT NULL,
  `images` JSON DEFAULT NULL,
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '0=待审核 1=可见 2=拒绝 3=删除',
  `audit_reason` VARCHAR(255) NOT NULL DEFAULT '',
  `like_count` BIGINT NOT NULL DEFAULT 0,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  INDEX `idx_campus_comment_post_created` (`post_id`, `status`, `is_deleted`, `created_at`, `id`),
  INDEX `idx_campus_comment_author` (`author_id`, `is_deleted`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园论坛评论';

CREATE TABLE IF NOT EXISTS `campus_forum_post_like` (
  `id` BIGINT NOT NULL,
  `post_id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_post_like_user` (`post_id`, `user_id`),
  INDEX `idx_campus_post_like_post` (`post_id`, `is_deleted`, `created_at`),
  INDEX `idx_campus_post_like_user` (`user_id`, `is_deleted`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园论坛帖子点赞';

CREATE TABLE IF NOT EXISTS `campus_audit_log` (
  `id` BIGINT NOT NULL,
  `target_type` VARCHAR(32) NOT NULL,
  `target_id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `provider` VARCHAR(32) NOT NULL,
  `result` VARCHAR(32) NOT NULL,
  `reason` VARCHAR(255) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  INDEX `idx_campus_audit_target` (`target_type`, `target_id`),
  INDEX `idx_campus_audit_user` (`user_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园内容审核记录';
