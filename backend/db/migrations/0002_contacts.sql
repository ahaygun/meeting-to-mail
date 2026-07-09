-- 0002_contacts.sql — kayıtlı alıcılar (kişi rehberi)

BEGIN;

CREATE TABLE IF NOT EXISTS contacts (
    id           BIGSERIAL PRIMARY KEY,
    email        TEXT        NOT NULL UNIQUE,
    name         TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_contacts_last_used ON contacts (last_used_at DESC);

COMMIT;
