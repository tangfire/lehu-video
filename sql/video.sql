CREATE database lehu_video_db;

use lehu_video_db;

CREATE TABLE `video` (
                         `id` bigint(20) NOT NULL AUTO_INCREMENT,
                         `user_id` bigint(20) DEFAULT NULL,
                         `title` varchar(20) DEFAULT NULL,
                         `description` varchar(50) DEFAULT NULL,
                         `video_url` varchar(255) DEFAULT NULL,
                         `cover_url` varchar(255) DEFAULT NULL,
                         `like_count` bigint(20) DEFAULT 0,
                         `comment_count` bigint(20) DEFAULT 0,
                         `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                         `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                         PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


CREATE TABLE `user` (
                        `id` bigint(20) NOT NULL AUTO_INCREMENT,
                        `account_id` bigint(20) DEFAULT NULL,
                        `mobile` varchar(20) DEFAULT NULL,
                        `email` varchar(50) DEFAULT NULL,
                        `name` varchar(50) DEFAULT NULL,
                        `avatar` varchar(255) DEFAULT NULL,
                        `background_image` varchar(255) DEFAULT NULL,
                        `signature` varchar(255) DEFAULT NULL,
                        `created_at` datetime(3) DEFAULT NULL,
                        `updated_at` datetime(3) DEFAULT NULL,
                        PRIMARY KEY (`id`),
                        UNIQUE KEY `idx_users_account_id` (`account_id`) USING BTREE,
                        UNIQUE KEY `idx_users_email` (`email`) USING BTREE,
                        UNIQUE KEY `idx_users_mobile` (`mobile`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=7240204103809514232 DEFAULT CHARSET=utf8mb4;

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
                                          target_type TINYINT NOT NULL COMMENT '点赞对象类型 1-视频 2-评论',
                                          target_id BIGINT NOT NULL COMMENT '点赞对象ID',
                                          favorite_type TINYINT NOT NULL COMMENT '点赞类型 1-点赞 2-踩',
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
                                         first_comments json NOT NULL COMMENT '最开始的x条子评论',
                                         is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
                                         created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                         updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                         INDEX `video_id_idx` (video_id, is_deleted),
                                         INDEX `user_id_idx` (user_id, is_deleted)
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

