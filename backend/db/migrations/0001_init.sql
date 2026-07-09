-- 0001_init.sql — Toplantı Kayıt & Özet Otomasyonu şeması
-- Tüm zaman damgaları UTC (timestamptz).

BEGIN;

-- sessions: kayıt öncesi konfig + durum makinesi
CREATE TABLE IF NOT EXISTS sessions (
    id                    UUID PRIMARY KEY,
    title                 TEXT        NOT NULL,
    status                TEXT        NOT NULL DEFAULT 'configuring',
    summary_style         TEXT        NOT NULL DEFAULT 'decisions_actions',
    send_policy           TEXT        NOT NULL DEFAULT 'immediate',   -- immediate | cancel_window
    cancel_window_seconds INT         NOT NULL DEFAULT 0,
    error_message         TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at            TIMESTAMPTZ,
    ended_at              TIMESTAMPTZ
);

-- session_recipients: oturumun alıcı e-postaları
CREATE TABLE IF NOT EXISTS session_recipients (
    id         BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    email      TEXT NOT NULL,
    UNIQUE (session_id, email)
);

-- session_participants: opsiyonel katılımcı isimleri (konuşmacı atfı için)
CREATE TABLE IF NOT EXISTS session_participants (
    id         BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    name       TEXT NOT NULL
);

-- audio_chunks: parça parça gelen ses
CREATE TABLE IF NOT EXISTS audio_chunks (
    id           BIGSERIAL PRIMARY KEY,
    session_id   UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    seq          INT  NOT NULL,
    storage_path TEXT NOT NULL,
    size_bytes   BIGINT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (session_id, seq)
);

-- transcripts: ASR çıktısı
CREATE TABLE IF NOT EXISTS transcripts (
    id         BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    provider   TEXT NOT NULL,
    language   TEXT NOT NULL DEFAULT 'tr',
    text       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- summaries: yapılandırılmış özet (çoklu satır — yeniden özetleme için)
CREATE TABLE IF NOT EXISTS summaries (
    id           BIGSERIAL PRIMARY KEY,
    session_id   UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    style        TEXT NOT NULL DEFAULT 'decisions_actions',
    content_json JSONB NOT NULL,
    content_text TEXT  NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- jobs: async iş kuyruğu (transcribe | summarize | send)
CREATE TABLE IF NOT EXISTS jobs (
    id          BIGSERIAL PRIMARY KEY,
    session_id  UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,                       -- transcribe | summarize | send
    status      TEXT NOT NULL DEFAULT 'pending',     -- pending | running | done | failed | cancelled
    run_at      TIMESTAMPTZ NOT NULL DEFAULT now(),  -- gönderim penceresini de yönetir
    attempts    INT  NOT NULL DEFAULT 0,
    last_error  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_jobs_claim ON jobs (status, run_at);

-- email_deliveries: gönderim logu (alıcı bazında durum)
CREATE TABLE IF NOT EXISTS email_deliveries (
    id           BIGSERIAL PRIMARY KEY,
    session_id   UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    recipient    TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',    -- pending | sent | failed
    provider_id  TEXT,
    error_message TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;
