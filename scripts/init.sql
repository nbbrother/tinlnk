-- TinLink 数据库初始化脚本
USE tinlink;

-- 创建分表存储过程
DELIMITER //
CREATE PROCEDURE create_url_tables()
BEGIN
    DECLARE i INT DEFAULT 0;
    WHILE i < 64 DO
        SET @sql = CONCAT('CREATE TABLE IF NOT EXISTS url_mapping_', LPAD(i, 2, '0'), ' (
            id BIGINT UNSIGNED PRIMARY KEY COMMENT ''Snowflake ID'',
            short_code VARCHAR(10) NOT NULL COMMENT ''短码'',
            long_url VARCHAR(2048) NOT NULL COMMENT ''原始URL'',
            user_id BIGINT UNSIGNED DEFAULT 0 COMMENT ''用户ID'',
            access_count BIGINT UNSIGNED DEFAULT 0 COMMENT ''访问次数'',
            expire_at DATETIME NOT NULL COMMENT ''过期时间'',
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT ''创建时间'',
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT ''更新时间'',
            UNIQUE KEY uk_short_code (short_code),
            KEY idx_user_id (user_id),
            KEY idx_expire_at (expire_at),
            KEY idx_created_at (created_at)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci');
        PREPARE stmt FROM @sql;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
        SET i = i + 1;
    END WHILE;
END //
DELIMITER ;

CALL create_url_tables();
DROP PROCEDURE IF EXISTS create_url_tables;

-- 统计表
CREATE TABLE IF NOT EXISTS url_stats (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    short_code VARCHAR(10) NOT NULL COMMENT '短码',
    stat_date DATE NOT NULL COMMENT '统计日期',
    pv BIGINT UNSIGNED DEFAULT 0 COMMENT '页面浏览量',
    uv BIGINT UNSIGNED DEFAULT 0 COMMENT '独立访客数',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_code_date (short_code, stat_date),
    KEY idx_stat_date (stat_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 访问日志表（可选，用于详细分析）
CREATE TABLE IF NOT EXISTS access_log (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    short_code VARCHAR(10) NOT NULL,
    ip VARCHAR(45) NOT NULL,
    user_agent VARCHAR(500),
    referer VARCHAR(2048),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    KEY idx_short_code (short_code),
    KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;