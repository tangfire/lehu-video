USE lehu_campus_db;

INSERT INTO `campus_ops_setting` (`setting_key`, `setting_value`, `updated_by`) VALUES
('audit_high_risk_words', '赌博,裸聊,诈骗,代考,代课,身份证,银行卡,毒品,买卖账号,刷单,套现', 0),
('audit_review_words', '加微信,兼职,引战,辱骂,曝光,挂人,联系方式,私聊,群号,二维码', 0)
ON DUPLICATE KEY UPDATE `setting_key` = `setting_key`;
