CREATE TABLE IF NOT EXISTS users (
    id         VARCHAR(64)  NOT NULL,
    username   VARCHAR(64)  NOT NULL,
    email      VARCHAR(128) NULL,
    phone      VARCHAR(32)  NULL,
    nickname   VARCHAR(64)  NULL,
    avatar     VARCHAR(255) NULL,
    status     VARCHAR(16)  NULL,
    created_at DATETIME     NULL,
    updated_at DATETIME     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_users_username (username),
    KEY idx_users_phone (phone)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;
