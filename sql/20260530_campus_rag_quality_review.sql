ALTER TABLE `campus_rag_query_log`
  ADD COLUMN `quality_label` VARCHAR(24) NOT NULL DEFAULT '' COMMENT 'good/needs_fix/wrong/unsafe' AFTER `error_message`,
  ADD COLUMN `quality_note` VARCHAR(500) NOT NULL DEFAULT '' AFTER `quality_label`,
  ADD COLUMN `reviewed_by` BIGINT NOT NULL DEFAULT 0 AFTER `quality_note`,
  ADD COLUMN `reviewed_at` DATETIME(3) DEFAULT NULL AFTER `reviewed_by`,
  ADD INDEX `idx_campus_rag_log_quality` (`quality_label`, `created_at`);
