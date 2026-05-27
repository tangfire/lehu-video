USE lehu_video_db;

-- Password hashes now support modern Argon2id PHC strings while keeping legacy MD5+salt rows readable.
ALTER TABLE account MODIFY COLUMN password VARCHAR(255) NOT NULL;
ALTER TABLE account MODIFY COLUMN salt VARCHAR(128) NOT NULL DEFAULT '';

-- Account uniqueness for login credentials.
ALTER TABLE account ADD UNIQUE KEY uk_account_mobile (mobile);
ALTER TABLE account ADD UNIQUE KEY uk_account_email (email);

-- Stable pagination and author profile queries.
ALTER TABLE video ADD INDEX idx_video_author_created (user_id, created_at, id);
ALTER TABLE video ADD INDEX idx_video_hot (created_at, like_count, comment_count, view_count);

-- Idempotent relationships and fast relation checks.
ALTER TABLE follow ADD UNIQUE KEY uk_follow_user_target_active (user_id, target_user_id, is_deleted);
ALTER TABLE collection_video ADD UNIQUE KEY uk_collection_video_active (collection_id, video_id, is_deleted);

-- Comment list pagination and child comment queries.
ALTER TABLE comment ADD INDEX idx_comment_video_parent_created (video_id, parent_id, is_deleted, created_at);
ALTER TABLE comment ADD INDEX idx_comment_parent_created (parent_id, is_deleted, created_at);
