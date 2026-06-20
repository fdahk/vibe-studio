CREATE TABLE IF NOT EXISTS sessions (
    id              VARCHAR(64)  NOT NULL,
    user_id         VARCHAR(64)  NOT NULL,
    token_hash      VARCHAR(64)  NOT NULL,
    prev_token_hash VARCHAR(64)  NULL,
    user_agent      VARCHAR(255) NULL,
    ip              VARCHAR(64)  NULL,
    expires_at      DATETIME     NOT NULL,
    revoked_at      DATETIME     NULL,
    last_used_at    DATETIME     NULL,
    created_at      DATETIME     NULL,
    updated_at      DATETIME     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_sessions_token_hash (token_hash),
    KEY idx_sessions_prev_token_hash (prev_token_hash),
    KEY idx_sessions_user_id (user_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;
