-- 0003_recipient_groups.sql — kayıtlı alıcı grupları (dağıtım listeleri)

BEGIN;

CREATE TABLE IF NOT EXISTS recipient_groups (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS recipient_group_members (
    id       BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES recipient_groups(id) ON DELETE CASCADE,
    email    TEXT   NOT NULL,
    UNIQUE (group_id, email)
);

COMMIT;
