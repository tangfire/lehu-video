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