CREATE TABLE IF NOT EXISTS identities (
    id           VARCHAR(64)  NOT NULL,
    user_id      VARCHAR(64)  NOT NULL,
    provider     VARCHAR(32)  NOT NULL,
    provider_uid VARCHAR(191) NOT NULL,
    secret       VARCHAR(255) NULL,
    created_at   DATETIME     NULL,
    updated_at   DATETIME     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_identities_provider_uid (provider, provider_uid),
    KEY idx_identities_user_id (user_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;
