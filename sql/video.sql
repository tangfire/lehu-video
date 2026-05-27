CREATE DATABASE IF NOT EXISTS lehu_video_db
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;

USE lehu_video_db;

CREATE TABLE IF NOT EXISTS `account` (
  `id` BIGINT NOT NULL,
  `mobile` VARCHAR(20) NOT NULL,
  `email` VARCHAR(100) NOT NULL,
  `password` VARCHAR(255) NOT NULL,
  `salt` VARCHAR(128) NOT NULL DEFAULT '',
  `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
  `active_mobile` VARCHAR(20) GENERATED ALWAYS AS (IF(`is_deleted` = 0, `mobile`, NULL)) STORED,
  `active_email` VARCHAR(100) GENERATED ALWAYS AS (IF(`is_deleted` = 0, `email`, NULL)) STORED,
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
