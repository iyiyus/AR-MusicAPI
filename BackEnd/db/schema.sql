-- 音乐API管理系统 数据库结构
-- MySQL 5.7+ / utf8mb4

CREATE DATABASE IF NOT EXISTS `music_admin`
  DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `music_admin`;

-- --------------------------------------------------------
-- 管理员表
-- --------------------------------------------------------
CREATE TABLE IF NOT EXISTS `admins` (
  `id`            INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `username`      VARCHAR(64)  NOT NULL COMMENT '登录账号',
  `password`      VARCHAR(255) NOT NULL COMMENT '密码哈希 pbkdf2$sha256$迭代次数$salt$hash',
  `nickname`      VARCHAR(64)  DEFAULT '' COMMENT '昵称',
  `avatar`        VARCHAR(500) DEFAULT '' COMMENT '头像',
  `roles`         VARCHAR(255) DEFAULT 'R_SUPER' COMMENT '角色，逗号分隔',
  `buttons`       VARCHAR(500) DEFAULT 'add,edit,delete,reset' COMMENT '按钮权限，逗号分隔',
  `status`        TINYINT(1)   DEFAULT 1 COMMENT '状态：1启用 0禁用',
  `last_login_at` DATETIME     DEFAULT NULL COMMENT '最后登录时间',
  `last_login_ip` VARCHAR(50)  DEFAULT '' COMMENT '最后登录IP',
  `created_at`    DATETIME     DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='管理员表';

-- --------------------------------------------------------
-- Cookie 池表
-- --------------------------------------------------------
CREATE TABLE IF NOT EXISTS `cookies` (
  `id`         INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `source`     VARCHAR(50)  NOT NULL COMMENT '音乐源：qqmusic/wyy/kugou/qishui',
  `cookie`     TEXT         NOT NULL COMMENT 'Cookie字符串',
  `remark`     VARCHAR(255) DEFAULT '' COMMENT '备注',
  `status`     TINYINT(1)   DEFAULT 1 COMMENT '状态：1启用 0禁用',
  `fail_count` INT          DEFAULT 0 COMMENT '连续失败次数',
  `call_count` INT          DEFAULT 0 COMMENT '调用次数',
  `last_used`  DATETIME     DEFAULT NULL COMMENT '最后使用时间',
  `created_at` DATETIME     DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_cookies_source` (`source`),
  KEY `idx_cookies_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Cookie池表';

-- --------------------------------------------------------
-- API 调用日志表
-- --------------------------------------------------------
CREATE TABLE IF NOT EXISTS `api_logs` (
  `id`             INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `source`         VARCHAR(50)  NOT NULL COMMENT '音乐源',
  `song_name`      VARCHAR(255) DEFAULT '' COMMENT '歌曲名',
  `singer`         VARCHAR(255) NOT NULL DEFAULT '' COMMENT '歌手',
  `song_id`        VARCHAR(120) NOT NULL DEFAULT '' COMMENT '歌曲ID',
  `album_img`      VARCHAR(500) DEFAULT '' COMMENT '专辑图片URL',
  `music_url`      VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '播放直链',
  `lyric_lrc`      MEDIUMTEXT NOT NULL COMMENT 'LRC歌词',
  `request_params` TEXT         COMMENT '请求参数JSON',
  `response_code`  INT          DEFAULT NULL COMMENT '响应码',
  `success`        TINYINT(1)   DEFAULT 0 COMMENT '是否成功：1成功 0失败',
  `error_msg`      TEXT         COMMENT '错误信息',
  `duration_ms`    INT          DEFAULT 0 COMMENT '耗时毫秒',
  `ip`             VARCHAR(50)  DEFAULT '' COMMENT 'IP地址',
  `user_agent`     TEXT         COMMENT '用户代理',
  `created_at`     DATETIME     DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_logs_source` (`source`),
  KEY `idx_logs_success` (`success`),
  KEY `idx_logs_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='API调用日志表';

-- --------------------------------------------------------
-- 系统配置表
-- --------------------------------------------------------
CREATE TABLE IF NOT EXISTS `settings` (
  `id`         INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `key_name`   VARCHAR(100) NOT NULL COMMENT '配置键名',
  `key_value`  TEXT         COMMENT '配置值',
  `updated_at` DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_key_name` (`key_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='系统配置表';

-- --------------------------------------------------------
-- 初始配置
-- --------------------------------------------------------
INSERT INTO `settings` (`key_name`, `key_value`) VALUES
  ('default_source',        'qqmusic'),
  ('site_name',             '音乐API管理系统'),
  ('site_icon',             ''),
  ('log_retention_days',    '30'),
  ('auto_switch_cookie',    '1'),
  ('cookie_fail_threshold', '3'),
  ('allow_public_api',      '1')
ON DUPLICATE KEY UPDATE `key_value` = VALUES(`key_value`);
