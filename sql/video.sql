CREATE DATABASE IF NOT EXISTS lehu_video_db
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;

USE lehu_video_db;
SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `account` (
  `id` BIGINT NOT NULL,
  `mobile` VARCHAR(20) NOT NULL,
  `email` VARCHAR(100) NOT NULL,
  `password` VARCHAR(255) NOT NULL,
  `salt` VARCHAR(128) NOT NULL DEFAULT '',
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `active_mobile` VARCHAR(20) GENERATED ALWAYS AS (IF(`is_deleted` = 0 AND `mobile` <> '', `mobile`, NULL)) STORED,
  `active_email` VARCHAR(100) GENERATED ALWAYS AS (IF(`is_deleted` = 0 AND `email` <> '', `email`, NULL)) STORED,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_account_mobile_active` (`active_mobile`),
  UNIQUE KEY `uk_account_email_active` (`active_email`),
  INDEX `idx_account_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `user` (
  `id` BIGINT NOT NULL,
  `account_id` BIGINT DEFAULT NULL,
  `mobile` VARCHAR(20) DEFAULT NULL,
  `email` VARCHAR(100) DEFAULT NULL,
  `name` VARCHAR(100) DEFAULT NULL,
  `nickname` VARCHAR(100) DEFAULT NULL,
  `avatar` VARCHAR(500) DEFAULT NULL,
  `background_image` VARCHAR(500) DEFAULT NULL,
  `signature` VARCHAR(500) DEFAULT NULL,
  `gender` INT DEFAULT 0,
  `follow_count` BIGINT NOT NULL DEFAULT 0,
  `follower_count` BIGINT NOT NULL DEFAULT 0,
  `be_liked_count` BIGINT NOT NULL DEFAULT 0,
  `work_count` BIGINT NOT NULL DEFAULT 0,
  `collection_count` BIGINT NOT NULL DEFAULT 0,
  `last_online_time` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  INDEX `idx_user_account_id` (`account_id`),
  INDEX `idx_user_mobile` (`mobile`),
  INDEX `idx_user_email` (`email`),
  INDEX `idx_user_nickname` (`nickname`),
  INDEX `idx_user_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `video` (
  `id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `title` VARCHAR(100) DEFAULT NULL,
  `description` VARCHAR(512) DEFAULT NULL,
  `video_url` VARCHAR(2048) DEFAULT NULL,
  `cover_url` VARCHAR(2048) DEFAULT NULL,
  `like_count` BIGINT NOT NULL DEFAULT 0,
  `comment_count` BIGINT NOT NULL DEFAULT 0,
  `collection_count` BIGINT NOT NULL DEFAULT 0,
  `view_count` BIGINT NOT NULL DEFAULT 0,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `idx_video_author_created` (`user_id`, `created_at`, `id`),
  INDEX `idx_video_created` (`created_at`, `id`),
  INDEX `idx_video_hot` (`created_at`, `like_count`, `comment_count`, `view_count`),
  INDEX `idx_video_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `follow` (
  `id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `target_user_id` BIGINT NOT NULL COMMENT '被关注的用户ID',
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_follow_user_target_active` (`user_id`, `target_user_id`, `is_deleted`),
  INDEX `idx_follow_user` (`user_id`, `is_deleted`, `updated_at`),
  INDEX `idx_follow_target` (`target_user_id`, `is_deleted`, `updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `favorite` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL,
  `target_type` TINYINT NOT NULL COMMENT '点赞对象类型 0=视频 1=评论',
  `target_id` BIGINT NOT NULL,
  `favorite_type` TINYINT NOT NULL COMMENT '点赞类型 0=点赞 1=踩',
  `delete_at` BIGINT NOT NULL DEFAULT 0 COMMENT '0 表示有效，非 0 表示软删除时间戳',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_favorite_user_target_active` (`user_id`, `target_id`, `target_type`, `favorite_type`, `delete_at`),
  INDEX `idx_favorite_user` (`user_id`, `target_type`, `delete_at`, `created_at`),
  INDEX `idx_favorite_target` (`target_type`, `target_id`, `favorite_type`, `delete_at`),
  INDEX `idx_favorite_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `comment` (
  `id` BIGINT NOT NULL,
  `video_id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL COMMENT '发表评论的用户ID',
  `parent_id` BIGINT NOT NULL DEFAULT 0 COMMENT '父评论ID，0 表示一级评论',
  `to_user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '回复的用户ID',
  `content` VARCHAR(512) NOT NULL,
  `like_count` BIGINT NOT NULL DEFAULT 0,
  `reply_count` BIGINT NOT NULL DEFAULT 0,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `idx_comment_video_parent_created` (`video_id`, `parent_id`, `is_deleted`, `created_at`),
  INDEX `idx_comment_parent_created` (`parent_id`, `is_deleted`, `created_at`),
  INDEX `idx_comment_user` (`user_id`, `is_deleted`, `created_at`),
  INDEX `idx_comment_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `collection` (
  `id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `title` VARCHAR(255) NOT NULL,
  `description` TEXT NOT NULL,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `idx_collection_user` (`user_id`, `is_deleted`, `created_at`),
  INDEX `idx_collection_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `collection_video` (
  `id` BIGINT NOT NULL,
  `collection_id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `video_id` BIGINT NOT NULL,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_collection_video_active` (`collection_id`, `video_id`, `is_deleted`),
  INDEX `idx_collection_video_collection` (`collection_id`, `is_deleted`, `created_at`),
  INDEX `idx_collection_video_video` (`video_id`, `is_deleted`, `updated_at`),
  INDEX `idx_collection_video_user` (`user_id`, `is_deleted`, `updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户收藏视频关系表';

CREATE TABLE IF NOT EXISTS `file` (
  `id` BIGINT NOT NULL,
  `domain_name` VARCHAR(100) NOT NULL,
  `biz_name` VARCHAR(100) NOT NULL,
  `hash` VARCHAR(255) NOT NULL,
  `file_size` BIGINT NOT NULL DEFAULT 0,
  `file_type` VARCHAR(255) NOT NULL,
  `uploaded` BOOLEAN NOT NULL DEFAULT FALSE,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `idx_file_created_at` (`created_at`),
  INDEX `idx_file_updated_at` (`updated_at`),
  INDEX `idx_file_hash` (`hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `file_shortvideo_short_video_hash_0` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_short_video_hash_1` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_short_video_hash_2` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_short_video_hash_3` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_short_video_hash_4` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_short_video_id_0` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_short_video_id_1` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_short_video_id_2` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_short_video_id_3` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_short_video_id_4` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_public_hash_0` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_public_hash_1` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_public_hash_2` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_public_hash_3` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_public_hash_4` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_public_id_0` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_public_id_1` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_public_id_2` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_public_id_3` LIKE `file`;
CREATE TABLE IF NOT EXISTS `file_shortvideo_public_id_4` LIKE `file`;

CREATE TABLE IF NOT EXISTS `group_info` (
  `id` BIGINT NOT NULL COMMENT '群聊ID',
  `name` VARCHAR(20) NOT NULL COMMENT '群名称',
  `notice` VARCHAR(500) DEFAULT NULL COMMENT '群公告',
  `member_cnt` INT DEFAULT 1 COMMENT '群人数',
  `owner_id` BIGINT NOT NULL COMMENT '群主ID',
  `add_mode` TINYINT DEFAULT 0 COMMENT '加群方式 0=直接 1=审核',
  `avatar` VARCHAR(255) DEFAULT NULL COMMENT '头像',
  `status` TINYINT DEFAULT 0 COMMENT '状态 0=正常 1=禁用 2=解散',
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  `is_deleted` TINYINT(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  INDEX `idx_group_info_owner` (`owner_id`),
  INDEX `idx_group_info_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='群聊信息表';

CREATE TABLE IF NOT EXISTS `group_member` (
  `id` BIGINT NOT NULL COMMENT '成员ID',
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `group_id` BIGINT NOT NULL COMMENT '群聊ID',
  `role` TINYINT DEFAULT 0 COMMENT '角色 0=普通成员 1=管理员 2=群主',
  `join_time` DATETIME NOT NULL COMMENT '加入时间',
  `is_deleted` TINYINT(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  INDEX `idx_group_member_group` (`group_id`),
  INDEX `idx_group_member_user` (`user_id`),
  INDEX `idx_group_member_group_user` (`group_id`, `user_id`, `is_deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='群成员表';

CREATE TABLE IF NOT EXISTS `group_apply` (
  `id` BIGINT NOT NULL COMMENT '申请ID',
  `user_id` BIGINT NOT NULL COMMENT '申请用户ID',
  `group_id` BIGINT NOT NULL COMMENT '群聊ID',
  `apply_reason` VARCHAR(200) DEFAULT NULL COMMENT '申请理由',
  `status` TINYINT DEFAULT 0 COMMENT '状态 0=待处理 1=已通过 2=已拒绝',
  `handler_id` BIGINT DEFAULT NULL COMMENT '处理人ID',
  `reply_msg` VARCHAR(200) DEFAULT NULL COMMENT '回复消息',
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  `is_deleted` TINYINT(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  INDEX `idx_group_apply_group` (`group_id`),
  INDEX `idx_group_apply_user` (`user_id`),
  INDEX `idx_group_apply_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='加群申请表';

CREATE TABLE IF NOT EXISTS `conversation` (
  `id` BIGINT NOT NULL COMMENT '会话ID',
  `type` TINYINT NOT NULL COMMENT '会话类型 0=单聊 1=群聊',
  `group_id` BIGINT DEFAULT NULL COMMENT '群ID，仅群聊有效',
  `name` VARCHAR(100) DEFAULT '' COMMENT '会话名称',
  `avatar` VARCHAR(500) DEFAULT '' COMMENT '会话头像',
  `last_message` TEXT COMMENT '最后一条消息内容',
  `last_msg_type` TINYINT DEFAULT NULL COMMENT '最后一条消息类型',
  `last_msg_time` DATETIME DEFAULT NULL COMMENT '最后一条消息时间',
  `member_count` BIGINT DEFAULT 1 COMMENT '成员数量',
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  `is_deleted` TINYINT(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  INDEX `idx_conversation_group` (`group_id`),
  INDEX `idx_conversation_last_msg_time` (`last_msg_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='会话主表';

CREATE TABLE IF NOT EXISTS `conversation_member` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `conversation_id` BIGINT NOT NULL COMMENT '会话ID',
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `type` TINYINT NOT NULL DEFAULT 0 COMMENT '成员类型 0=普通成员 1=管理员 2=群主',
  `unread_count` INT DEFAULT 0 COMMENT '未读消息数',
  `last_read_msg_id` BIGINT DEFAULT 0 COMMENT '最后已读消息ID',
  `is_pinned` TINYINT(1) DEFAULT 0 COMMENT '是否置顶',
  `is_muted` TINYINT(1) DEFAULT 0 COMMENT '是否免打扰',
  `join_time` DATETIME DEFAULT NULL COMMENT '加入时间',
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  `is_deleted` TINYINT(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_conversation_user` (`conversation_id`, `user_id`),
  INDEX `idx_conversation_member_user` (`user_id`),
  INDEX `idx_conversation_member_unread` (`unread_count`),
  INDEX `idx_conversation_member_conversation` (`conversation_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='会话成员表';

CREATE TABLE IF NOT EXISTS `message` (
  `id` BIGINT NOT NULL COMMENT '消息ID',
  `sender_id` BIGINT NOT NULL COMMENT '发送者ID',
  `receiver_id` BIGINT NOT NULL COMMENT '接收者ID，用户ID或群ID',
  `conversation_id` BIGINT DEFAULT NULL COMMENT '会话ID',
  `conv_type` TINYINT NOT NULL COMMENT '会话类型 0=单聊 1=群聊',
  `msg_type` TINYINT NOT NULL COMMENT '消息类型 0=文本 1=图片 2=语音 3=视频 4=文件 99=系统',
  `content` JSON NOT NULL COMMENT '消息内容',
  `status` TINYINT DEFAULT 0 COMMENT '消息状态 0=发送中 1=已发送 2=已送达 3=已读 4=已撤回 99=失败',
  `is_recalled` TINYINT(1) DEFAULT 0 COMMENT '是否已撤回',
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  `is_deleted` TINYINT(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  INDEX `idx_message_sender` (`sender_id`),
  INDEX `idx_message_receiver` (`receiver_id`),
  INDEX `idx_message_conv_type` (`conv_type`),
  INDEX `idx_message_created_at` (`created_at`),
  INDEX `idx_message_conversation` (`conversation_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='消息表';

CREATE TABLE IF NOT EXISTS `user_online_status` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `online_status` TINYINT NOT NULL DEFAULT 0 COMMENT '在线状态 0=离线 1=在线 2=忙碌 3=离开',
  `device_type` VARCHAR(20) DEFAULT '' COMMENT '设备类型 web/ios/android',
  `last_online_time` DATETIME NOT NULL COMMENT '最后在线时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_online_status_user` (`user_id`),
  INDEX `idx_user_online_status` (`online_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户在线状态表';

CREATE TABLE IF NOT EXISTS `friend_relation` (
  `id` BIGINT NOT NULL COMMENT '主键ID',
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `friend_id` BIGINT NOT NULL COMMENT '好友ID',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态 1=好友 2=已删除 3=拉黑',
  `remark` VARCHAR(100) DEFAULT '' COMMENT '备注',
  `group_name` VARCHAR(50) DEFAULT '' COMMENT '分组名称',
  `is_following` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否关注好友',
  `is_follower` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否被好友关注',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_friend_relation_user_friend` (`user_id`, `friend_id`),
  INDEX `idx_friend_relation_user` (`user_id`),
  INDEX `idx_friend_relation_friend` (`friend_id`),
  INDEX `idx_friend_relation_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='好友关系表';

CREATE TABLE IF NOT EXISTS `friend_apply` (
  `id` BIGINT NOT NULL COMMENT '申请ID',
  `applicant_id` BIGINT NOT NULL COMMENT '申请人ID',
  `receiver_id` BIGINT NOT NULL COMMENT '接收人ID',
  `apply_reason` VARCHAR(200) DEFAULT '' COMMENT '申请理由',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态 0=待处理 1=已同意 2=已拒绝',
  `handled_at` DATETIME DEFAULT NULL COMMENT '处理时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_friend_apply_applicant_receiver` (`applicant_id`, `receiver_id`),
  INDEX `idx_friend_apply_receiver_status` (`receiver_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='好友申请表';

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

CREATE TABLE IF NOT EXISTS `campus_timetable_course` (
  `id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `term` VARCHAR(32) NOT NULL,
  `course_name` VARCHAR(120) NOT NULL,
  `teacher` VARCHAR(80) NOT NULL DEFAULT '',
  `classroom` VARCHAR(120) NOT NULL DEFAULT '',
  `weekday` TINYINT NOT NULL COMMENT '1=周一 7=周日',
  `start_section` TINYINT NOT NULL,
  `end_section` TINYINT NOT NULL,
  `start_week` TINYINT NOT NULL DEFAULT 1,
  `end_week` TINYINT NOT NULL DEFAULT 20,
  `week_parity` TINYINT NOT NULL DEFAULT 0 COMMENT '0=每周 1=单周 2=双周',
  `source` VARCHAR(32) NOT NULL DEFAULT 'educational_system',
  `source_course_id` VARCHAR(128) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  INDEX `idx_campus_timetable_user_term` (`user_id`, `term`, `weekday`, `start_section`),
  INDEX `idx_campus_timetable_source` (`source`, `source_course_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园个人课表课程';

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
  `is_official` BOOLEAN NOT NULL DEFAULT FALSE COMMENT '官方/运营内容',
  `is_featured` BOOLEAN NOT NULL DEFAULT FALSE COMMENT '精选推荐',
  `is_pinned` BOOLEAN NOT NULL DEFAULT FALSE COMMENT '首页置顶',
  `sort_weight` INT NOT NULL DEFAULT 0 COMMENT '运营排序权重',
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
  INDEX `idx_campus_post_type` (`post_type`, `status`, `is_deleted`, `created_at`),
  INDEX `idx_campus_post_ops_sort` (`status`, `is_deleted`, `is_pinned`, `is_featured`, `sort_weight`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园社区笔记';

CREATE TABLE IF NOT EXISTS `campus_forum_comment` (
  `id` BIGINT NOT NULL,
  `post_id` BIGINT NOT NULL,
  `parent_id` BIGINT NOT NULL DEFAULT 0 COMMENT '一级评论 ID，0 表示根评论',
  `reply_to_comment_id` BIGINT NOT NULL DEFAULT 0 COMMENT '回复的评论 ID',
  `reply_to_user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '回复的用户 ID',
  `author_id` BIGINT NOT NULL,
  `content` VARCHAR(1000) NOT NULL,
  `images` JSON DEFAULT NULL,
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '0=待审核 1=可见 2=拒绝 3=删除',
  `audit_reason` VARCHAR(255) NOT NULL DEFAULT '',
  `like_count` BIGINT NOT NULL DEFAULT 0,
  `reply_count` BIGINT NOT NULL DEFAULT 0,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  INDEX `idx_campus_comment_post_created` (`post_id`, `status`, `is_deleted`, `created_at`, `id`),
  INDEX `idx_campus_comment_parent_created` (`parent_id`, `status`, `is_deleted`, `created_at`, `id`),
  INDEX `idx_campus_comment_author` (`author_id`, `is_deleted`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园论坛评论';

CREATE TABLE IF NOT EXISTS `campus_forum_comment_like` (
  `id` BIGINT NOT NULL,
  `comment_id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_comment_like_user` (`comment_id`, `user_id`),
  INDEX `idx_campus_comment_like_comment` (`comment_id`, `is_deleted`, `created_at`),
  INDEX `idx_campus_comment_like_user` (`user_id`, `is_deleted`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园论坛评论点赞';

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

CREATE TABLE IF NOT EXISTS `campus_forum_post_collection` (
  `id` BIGINT NOT NULL,
  `post_id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_post_collection_user` (`post_id`, `user_id`),
  INDEX `idx_campus_post_collection_post` (`post_id`, `is_deleted`, `created_at`),
  INDEX `idx_campus_post_collection_user` (`user_id`, `is_deleted`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园社区笔记收藏';

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

CREATE TABLE IF NOT EXISTS `campus_notification` (
  `id` BIGINT NOT NULL,
  `recipient_id` BIGINT NOT NULL COMMENT '接收用户',
  `actor_id` BIGINT NOT NULL DEFAULT 0 COMMENT '触发用户，系统通知为运营用户或0',
  `event_type` VARCHAR(32) NOT NULL COMMENT 'comment/reply/post_like/post_collect/comment_like/system',
  `target_type` VARCHAR(32) NOT NULL DEFAULT '',
  `target_id` BIGINT NOT NULL DEFAULT 0,
  `dedupe_key` VARCHAR(191) DEFAULT NULL COMMENT '互动通知幂等键，系统通知为空',
  `title` VARCHAR(120) NOT NULL DEFAULT '',
  `content` VARCHAR(600) NOT NULL DEFAULT '',
  `link_page` VARCHAR(64) NOT NULL DEFAULT '',
  `link_params` JSON DEFAULT NULL,
  `read_at` DATETIME(3) DEFAULT NULL,
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_notification_dedupe` (`dedupe_key`),
  INDEX `idx_campus_notification_user_created` (`recipient_id`, `is_deleted`, `created_at`, `id`),
  INDEX `idx_campus_notification_user_unread` (`recipient_id`, `read_at`, `is_deleted`, `created_at`),
  INDEX `idx_campus_notification_event` (`event_type`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园站内消息通知';

CREATE TABLE IF NOT EXISTS `campus_notification_outbox` (
  `id` BIGINT NOT NULL,
  `recipient_id` BIGINT NOT NULL DEFAULT 0 COMMENT '互动通知接收用户，系统群发为0',
  `actor_id` BIGINT NOT NULL DEFAULT 0 COMMENT '触发用户或运营用户',
  `event_type` VARCHAR(32) NOT NULL COMMENT 'comment/reply/post_like/post_collect/comment_like/system',
  `target_type` VARCHAR(32) NOT NULL DEFAULT '',
  `target_id` BIGINT NOT NULL DEFAULT 0,
  `dedupe_key` VARCHAR(191) DEFAULT NULL COMMENT '投递幂等键',
  `title` VARCHAR(120) NOT NULL DEFAULT '',
  `content` VARCHAR(600) NOT NULL DEFAULT '',
  `link_page` VARCHAR(64) NOT NULL DEFAULT '',
  `link_params` JSON DEFAULT NULL,
  `audience` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '系统通知范围，v1=all_users',
  `status` VARCHAR(24) NOT NULL DEFAULT 'pending' COMMENT 'pending/processing/done/failed',
  `retry_count` INT NOT NULL DEFAULT 0,
  `next_retry_at` DATETIME(3) DEFAULT NULL,
  `locked_until` DATETIME(3) DEFAULT NULL,
  `last_error` VARCHAR(600) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `processed_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_notification_outbox_dedupe` (`dedupe_key`),
  INDEX `idx_campus_notification_outbox_status_next` (`status`, `next_retry_at`, `locked_until`, `id`),
  INDEX `idx_campus_notification_outbox_created` (`created_at`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园通知可靠投递任务';

CREATE TABLE IF NOT EXISTS `campus_access_log` (
  `id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '游客为0',
  `ip` VARCHAR(64) NOT NULL DEFAULT '',
  `method` VARCHAR(12) NOT NULL DEFAULT '',
  `path` VARCHAR(255) NOT NULL DEFAULT '',
  `status_code` INT NOT NULL DEFAULT 0,
  `duration_ms` BIGINT NOT NULL DEFAULT 0,
  `user_agent` VARCHAR(512) NOT NULL DEFAULT '',
  `rate_limited` BOOLEAN NOT NULL DEFAULT FALSE,
  `blocked` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  INDEX `idx_campus_access_created` (`created_at`),
  INDEX `idx_campus_access_ip_created` (`ip`, `created_at`),
  INDEX `idx_campus_access_path_created` (`path`, `created_at`),
  INDEX `idx_campus_access_status_created` (`status_code`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园接口访问日志';

CREATE TABLE IF NOT EXISTS `campus_ip_block` (
  `id` BIGINT NOT NULL,
  `ip` VARCHAR(64) NOT NULL,
  `reason` VARCHAR(255) NOT NULL DEFAULT '',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1=生效 0=解除',
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campus_ip_block_ip` (`ip`),
  INDEX `idx_campus_ip_block_status` (`status`, `updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园接口 IP 封禁';

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

CREATE TABLE IF NOT EXISTS `campus_operator` (
  `user_id` BIGINT NOT NULL,
  `role` VARCHAR(24) NOT NULL DEFAULT 'operator' COMMENT 'operator/admin',
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`user_id`),
  INDEX `idx_campus_operator_role` (`role`, `is_deleted`, `updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园运营后台权限';

CREATE TABLE IF NOT EXISTS `campus_event` (
  `id` BIGINT NOT NULL,
  `user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '游客为0',
  `event_type` VARCHAR(32) NOT NULL COMMENT 'visit/share/login/post_create/comment_create/like/collect',
  `page` VARCHAR(64) NOT NULL DEFAULT '',
  `target_type` VARCHAR(32) NOT NULL DEFAULT '',
  `target_id` BIGINT NOT NULL DEFAULT 0,
  `channel` VARCHAR(64) NOT NULL DEFAULT '',
  `extra` JSON DEFAULT NULL,
  `user_agent` VARCHAR(512) NOT NULL DEFAULT '',
  `ip` VARCHAR(64) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  INDEX `idx_campus_event_type_created` (`event_type`, `created_at`),
  INDEX `idx_campus_event_user_created` (`user_id`, `created_at`),
  INDEX `idx_campus_event_target_created` (`target_type`, `target_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='校园小程序轻量行为埋点';
