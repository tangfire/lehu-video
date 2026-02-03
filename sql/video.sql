CREATE database lehu_video_db;

use lehu_video_db;

CREATE TABLE `video` (
                         `id` bigint(20) NOT NULL AUTO_INCREMENT,
                         `user_id` bigint(20) DEFAULT NULL,
                         `title` varchar(20) DEFAULT NULL,
                         `description` varchar(50) DEFAULT NULL,
                         `video_url` varchar(2048) DEFAULT NULL,
                         `cover_url` varchar(2048) DEFAULT NULL,
                         `like_count` bigint(20) DEFAULT 0,
                         `comment_count` bigint(20) DEFAULT 0,
                         `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                         `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                         PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- 创建新表
CREATE TABLE `user` (
                            `id` bigint(20) NOT NULL AUTO_INCREMENT,
                            `account_id` bigint(20) DEFAULT NULL,
                            `mobile` varchar(20) DEFAULT NULL,
                            `email` varchar(100) DEFAULT NULL,
                            `name` varchar(100) DEFAULT NULL,
                            `nickname` varchar(100) DEFAULT NULL,
                            `avatar` varchar(500) DEFAULT NULL,
                            `background_image` varchar(500) DEFAULT NULL,
                            `signature` varchar(500) DEFAULT NULL,
                            `gender` int(11) DEFAULT '0',
                            `follow_count` bigint(20) DEFAULT '0',
                            `follower_count` bigint(20) DEFAULT '0',
                            `total_favorited` bigint(20) DEFAULT '0',
                            `work_count` bigint(20) DEFAULT '0',
                            `favorite_count` bigint(20) DEFAULT '0',
                            `created_at` datetime(3) DEFAULT NULL,
                            `updated_at` datetime(3) DEFAULT NULL,
                            PRIMARY KEY (`id`),
                            KEY `idx_account_id` (`account_id`),
                            KEY `idx_mobile` (`mobile`),
                            KEY `idx_email` (`email`),
                            KEY `idx_nickname` (`nickname`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;





CREATE TABLE IF NOT EXISTS account (
                                       `id` BIGINT NOT NULL,
                                       `mobile` VARCHAR(20) NOT NULL,
                                       `email` VARCHAR(100) NOT NULL,
                                       `password` VARCHAR(64) NOT NULL,
                                       `salt` VARCHAR(64) NOT NULL,
                                       `is_deleted` BOOLEAN NOT NULL DEFAULT FALSE,
                                       `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                       `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                       PRIMARY KEY (`id`),
                                       INDEX `account_mobile_idx` (`mobile`),
                                       INDEX `account_email_idx` (`email`)
)ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;




CREATE TABLE IF NOT EXISTS `follow` (
                                        id BIGINT NOT NULL,
                                        `user_id` BIGINT NOT NULL,
                                        target_user_id BIGINT NOT NULL COMMENT '被关注的用户id',
                                        is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
                                        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                        INDEX `user_id_idx` (`user_id`, `target_user_id`, `is_deleted`),
                                        PRIMARY KEY(`id`)
);


CREATE TABLE IF NOT EXISTS `favorite` (
                                          id BIGINT PRIMARY KEY COMMENT '主键ID',
                                          user_id BIGINT NOT NULL COMMENT '用户ID',
                                          target_type TINYINT NOT NULL COMMENT '点赞对象类型 0-视频 1-评论',
                                          target_id BIGINT NOT NULL COMMENT '点赞对象ID',
                                          favorite_type TINYINT NOT NULL COMMENT '点赞类型 0-点赞 1-踩',
                                          is_deleted BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否删除',
                                          created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
                                          updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    -- 核心：条件唯一索引（仅对未删除的记录建立唯一约束）
    -- MySQL 8.0+ 支持函数索引
                                          UNIQUE INDEX `uniq_user_target_active` (
                                                                                  user_id,
                                                                                  target_id,
                                                                                  target_type,
                                                                                  favorite_type,
                                              (CASE WHEN is_deleted = FALSE THEN 1 END)
                                              ),

    -- 查询索引
                                          INDEX `idx_user_target_type` (user_id, target_type, is_deleted),
                                          INDEX `idx_target` (target_type, target_id, is_deleted),
                                          INDEX `idx_created_at` (created_at)
) COMMENT='用户收藏表';



CREATE TABLE IF NOT EXISTS `comment` (
                                         id BIGINT PRIMARY KEY,
                                         video_id BIGINT NOT NULL,
                                         `user_id` BIGINT NOT NULL COMMENT '发表评论的用户id',
                                         parent_id BIGINT DEFAULT NULL COMMENT '父评论id',
                                         to_user_id BIGINT DEFAULT NULL COMMENT '评论所回复的用户id',
                                         content varchar(512) NOT NULL COMMENT '评论内容',
                                         is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
                                         created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                         updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                         INDEX `video_id_idx` (video_id, is_deleted),
                                         INDEX `user_id_idx` (`user_id`, is_deleted)
);


CREATE TABLE IF NOT EXISTS `file` (
                                    id BIGINT PRIMARY KEY,
                                    domain_name VARCHAR(100) NOT NULL,
                                    biz_name VARCHAR(100) NOT NULL,
                                    hash VARCHAR(255) NOT NULL,
                                    file_size BIGINT NOT NULL DEFAULT 0,
                                    file_type VARCHAR(255) NOT NULL,
                                    uploaded BOOLEAN NOT NULL DEFAULT FALSE,
                                    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
                                    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                    INDEX `create_time_idx` (`created_at`),
                                    INDEX `update_time_idx` (`updated_at`),
                                    INDEX `hash_idx` (`hash`)
);


-- 创建短视频相关的分表（共5个hash分表 + 5个id分表）
CREATE TABLE `file_shortvideo_short_video_hash_0` LIKE `file`;
CREATE TABLE `file_shortvideo_short_video_hash_1` LIKE `file`;
CREATE TABLE `file_shortvideo_short_video_hash_2` LIKE `file`;
CREATE TABLE `file_shortvideo_short_video_hash_3` LIKE `file`;
CREATE TABLE `file_shortvideo_short_video_hash_4` LIKE `file`;

CREATE TABLE `file_shortvideo_short_video_id_0` LIKE `file`;
CREATE TABLE `file_shortvideo_short_video_id_1` LIKE `file`;
CREATE TABLE `file_shortvideo_short_video_id_2` LIKE `file`;
CREATE TABLE `file_shortvideo_short_video_id_3` LIKE `file`;
CREATE TABLE `file_shortvideo_short_video_id_4` LIKE `file`;

-- 创建公共文件相关的分表
CREATE TABLE `file_shortvideo_public_hash_0` LIKE `file`;
CREATE TABLE `file_shortvideo_public_hash_1` LIKE `file`;
CREATE TABLE `file_shortvideo_public_hash_2` LIKE `file`;
CREATE TABLE `file_shortvideo_public_hash_3` LIKE `file`;
CREATE TABLE `file_shortvideo_public_hash_4` LIKE `file`;

CREATE TABLE `file_shortvideo_public_id_0` LIKE `file`;
CREATE TABLE `file_shortvideo_public_id_1` LIKE `file`;
CREATE TABLE `file_shortvideo_public_id_2` LIKE `file`;
CREATE TABLE `file_shortvideo_public_id_3` LIKE `file`;
CREATE TABLE `file_shortvideo_public_id_4` LIKE `file`;



CREATE TABLE IF NOT EXISTS `collection` (
                                            id BIGINT PRIMARY KEY,
                                            `user_id` BIGINT NOT NULL,
                                            title VARCHAR(255) NOT NULL,
                                            description TEXT NOT NULL,
                                            is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
                                            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                            updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                            INDEX `user_id_idx` (`user_id`, `is_deleted`),
                                            INDEX `create_time_idx` (`created_at`),
                                            INDEX `update_time_idx` (`updated_at`)
);


CREATE TABLE IF NOT EXISTS `collection_video` (
                                                  id BIGINT PRIMARY KEY,
                                                  collection_id BIGINT NOT NULL,
                                                  user_id BIGINT NOT NULL,
                                                  video_id BIGINT NOT NULL,
                                                  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
                                                  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                                  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                                  INDEX `collection_id_idx` (`collection_id`, `is_deleted`)
);


-- 创建群聊信息表
CREATE TABLE `group_info` (
                              `id` bigint(20) NOT NULL COMMENT '群聊ID',
                              `name` varchar(20) NOT NULL COMMENT '群名称',
                              `notice` varchar(500) DEFAULT NULL COMMENT '群公告',
                              `member_cnt` int(11) DEFAULT '1' COMMENT '群人数',
                              `owner_id` bigint(20) NOT NULL COMMENT '群主ID',
                              `add_mode` tinyint(4) DEFAULT '0' COMMENT '加群方式，0.直接，1.审核',
                              `avatar` varchar(255) DEFAULT NULL COMMENT '头像',
                              `status` tinyint(4) DEFAULT '0' COMMENT '状态，0.正常，1.禁用，2.解散',
                              `created_at` datetime NOT NULL COMMENT '创建时间',
                              `updated_at` datetime NOT NULL COMMENT '更新时间',
                              `is_deleted` tinyint(1) NOT NULL DEFAULT '0',
                              PRIMARY KEY (`id`),
                              KEY `idx_owner_id` (`owner_id`),
                              KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群聊信息表';

-- 创建群成员表
CREATE TABLE `group_member` (
                                `id` bigint(20) NOT NULL COMMENT '成员ID',
                                `user_id` bigint(20) NOT NULL COMMENT '用户ID',
                                `group_id` bigint(20) NOT NULL COMMENT '群聊ID',
                                `role` tinyint(4) DEFAULT '0' COMMENT '角色，0.普通成员，1.管理员，2.群主',
                                `join_time` datetime NOT NULL COMMENT '加入时间',
                                `is_deleted` tinyint(1) NOT NULL DEFAULT '0',
                                PRIMARY KEY (`id`),
                                KEY `idx_group_id` (`group_id`),
                                KEY `idx_user_id` (`user_id`),
                                KEY `idx_group_user` (`group_id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群成员表';

-- 创建加群申请表
CREATE TABLE `group_apply` (
                               `id` bigint(20) NOT NULL COMMENT '申请ID',
                               `user_id` bigint(20) NOT NULL COMMENT '申请用户ID',
                               `group_id` bigint(20) NOT NULL COMMENT '群聊ID',
                               `apply_reason` varchar(200) DEFAULT NULL COMMENT '申请理由',
                               `status` tinyint(4) DEFAULT '0' COMMENT '状态，0.待处理，1.已通过，2.已拒绝',
                               `handler_id` bigint(20) DEFAULT NULL COMMENT '处理人ID',
                               `reply_msg` varchar(200) DEFAULT NULL COMMENT '回复消息',
                               `created_at` datetime NOT NULL COMMENT '创建时间',
                               `updated_at` datetime NOT NULL COMMENT '更新时间',
                               `is_deleted` tinyint(1) NOT NULL DEFAULT '0',
                               PRIMARY KEY (`id`),
                               KEY `idx_group_id` (`group_id`),
                               KEY `idx_user_id` (`user_id`),
                               KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='加群申请表';


-- 消息表
CREATE TABLE `message` (
                           `id` bigint(20) NOT NULL COMMENT '消息ID',
                           `sender_id` bigint(20) NOT NULL COMMENT '发送者ID',
                           `receiver_id` bigint(20) NOT NULL COMMENT '接收者ID（用户ID或群ID）',
                           `conversation_id` bigint(20) DEFAULT NULL COMMENT '会话ID',
                           `conv_type` tinyint(4) NOT NULL COMMENT '会话类型 0:单聊 1:群聊',
                           `msg_type` tinyint(4) NOT NULL COMMENT '消息类型 0:文本 1:图片 2:语音 3:视频 4:文件 99:系统',
                           `content` json NOT NULL COMMENT '消息内容',
                           `status` tinyint(4) DEFAULT '0' COMMENT '消息状态 0:发送中 1:已发送 2:已送达 3:已读 4:已撤回 99:失败',
                           `is_recalled` tinyint(1) DEFAULT '0' COMMENT '是否已撤回',
                           `created_at` datetime NOT NULL COMMENT '创建时间',
                           `updated_at` datetime NOT NULL COMMENT '更新时间',
                           `is_deleted` tinyint(1) NOT NULL DEFAULT '0',
                           PRIMARY KEY (`id`),
                           KEY `idx_sender_id` (`sender_id`),
                           KEY `idx_receiver_id` (`receiver_id`),
                           KEY `idx_conv_type` (`conv_type`),
                           KEY `idx_created_at` (`created_at`),
                           KEY `idx_conversation_id` (`conversation_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='消息表';



CREATE TABLE `conversation` (
                                `id` bigint(20) NOT NULL COMMENT '会话ID（主键）',
                                `type` tinyint(4) NOT NULL COMMENT '会话类型 0:单聊 1:群聊',
                                `group_id` bigint(20) DEFAULT NULL COMMENT '群ID（仅群聊有效）',
                                `name` varchar(100) DEFAULT '' COMMENT '会话名称（群聊时用）',
                                `avatar` varchar(500) DEFAULT '' COMMENT '会话头像',
                                `last_message` text COMMENT '最后一条消息内容',
                                `last_msg_type` tinyint(4) DEFAULT NULL COMMENT '最后一条消息类型',
                                `last_msg_time` datetime DEFAULT NULL COMMENT '最后一条消息时间',
                                `member_count` int(11) DEFAULT '1' COMMENT '成员数量',
                                `created_at` datetime NOT NULL COMMENT '创建时间',
                                `updated_at` datetime NOT NULL COMMENT '更新时间',
                                `is_deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除',
                                PRIMARY KEY (`id`),
                                UNIQUE KEY `uk_single_chat` (`type`, `target_id`),
                                UNIQUE KEY `uk_group_chat` (`type`, `group_id`),
                                KEY `idx_type_target` (`type`, `target_id`),
                                KEY `idx_group_id` (`group_id`),
                                KEY `idx_last_msg_time` (`last_msg_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='会话主表';

-- 2. 创建会话成员表
CREATE TABLE `conversation_member` (
                                       `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
                                       `conversation_id` bigint(20) NOT NULL COMMENT '会话ID',
                                       `user_id` bigint(20) NOT NULL COMMENT '用户ID',
                                       `type` tinyint(4) NOT NULL DEFAULT '0' COMMENT '成员类型 0:普通成员 1:管理员 2:群主',
                                       `unread_count` int(11) DEFAULT '0' COMMENT '未读消息数',
                                       `last_read_msg_id` bigint(20) DEFAULT '0' COMMENT '最后已读消息ID',
                                       `is_pinned` tinyint(1) DEFAULT '0' COMMENT '是否置顶',
                                       `is_muted` tinyint(1) DEFAULT '0' COMMENT '是否免打扰',
                                       `join_time` datetime DEFAULT NULL COMMENT '加入时间',
                                       `created_at` datetime NOT NULL COMMENT '创建时间',
                                       `updated_at` datetime NOT NULL COMMENT '更新时间',
                                       `is_deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除',
                                       PRIMARY KEY (`id`),
                                       UNIQUE KEY `uk_conversation_user` (`conversation_id`, `user_id`),
                                       KEY `idx_user_id` (`user_id`),
                                       KEY `idx_unread_count` (`unread_count`),
                                       KEY `idx_conversation_id` (`conversation_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='会话成员表';



-- 用户在线状态表（单独表）
CREATE TABLE `user_online_status` (
                                      `id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
                                      `user_id` bigint(20) NOT NULL COMMENT '用户ID',
                                      `online_status` tinyint(4) NOT NULL DEFAULT '0' COMMENT '在线状态：0=离线，1=在线，2=忙碌，3=离开',
                                      `device_type` varchar(20) DEFAULT '' COMMENT '设备类型：web/ios/android',
                                      `last_online_time` datetime NOT NULL COMMENT '最后在线时间',
                                      `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                      `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                      PRIMARY KEY (`id`),
                                      UNIQUE KEY `uk_user_id` (`user_id`),
                                      KEY `idx_online_status` (`online_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户在线状态表';

-- 好友关系表（优化版）
CREATE TABLE `friend_relation` (
                                   `id` bigint(20) NOT NULL  COMMENT '主键ID',
                                   `user_id` bigint(20) NOT NULL COMMENT '用户ID',
                                   `friend_id` bigint(20) NOT NULL COMMENT '好友ID',
                                   `status` tinyint(4) NOT NULL DEFAULT '1' COMMENT '状态：1=好友，2=已删除，3=拉黑',
                                   `remark` varchar(100) DEFAULT '' COMMENT '备注',
                                   `group_name` varchar(50) DEFAULT '' COMMENT '分组名称',
                                   `is_following` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否关注好友',
                                   `is_follower` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否被好友关注',
                                   `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                   `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                   PRIMARY KEY (`id`),
                                   UNIQUE KEY `uk_user_friend` (`user_id`, `friend_id`),
                                   KEY `idx_user_id` (`user_id`),
                                   KEY `idx_friend_id` (`friend_id`),
                                   KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='好友关系表';

-- 好友申请表
CREATE TABLE `friend_apply` (
                                `id` bigint(20) NOT NULL COMMENT 'ID',
                                `applicant_id` bigint(20) NOT NULL COMMENT '申请人ID',
                                `receiver_id` bigint(20) NOT NULL COMMENT '接收人ID',
                                `apply_reason` varchar(200) DEFAULT '' COMMENT '申请理由',
                                `status` tinyint(4) NOT NULL DEFAULT '0' COMMENT '状态：0=待处理，1=已同意，2=已拒绝',
                                `handled_at` datetime DEFAULT NULL COMMENT '处理时间',
                                `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                PRIMARY KEY (`id`),
                                UNIQUE KEY `uk_applicant_receiver` (`applicant_id`, `receiver_id`),
                                KEY `idx_receiver_status` (`receiver_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='好友申请表';