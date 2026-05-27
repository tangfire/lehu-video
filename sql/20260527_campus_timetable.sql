USE lehu_video_db;

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
