package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"meeting-to-mail/internal/domain"
)

// PG, Store arayüzünün Postgres implementasyonu.
type PG struct {
	pool *pgxpool.Pool
}

// NewPG bir bağlantı havuzu açar ve erişilebilirliği doğrular.
func NewPG(ctx context.Context, dsn string) (*PG, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return &PG{pool: pool}, nil
}

// Close havuzu kapatır.
func (p *PG) Close() { p.pool.Close() }

// Pool, migration çalıştırıcı gibi düşük seviye erişim isteyenler için.
func (p *PG) Pool() *pgxpool.Pool { return p.pool }

// --- Sessions ---

func (p *PG) CreateSession(ctx context.Context, s *domain.Session) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO sessions (id, title, status, summary_style, send_policy, cancel_window_seconds)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`,
		s.ID, s.Title, s.Status, s.SummaryStyle, s.SendPolicy, s.CancelWindowSeconds,
	).Scan(&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return err
	}

	for _, email := range s.Recipients {
		if _, err := tx.Exec(ctx,
			`INSERT INTO session_recipients (session_id, email) VALUES ($1, $2)
			 ON CONFLICT (session_id, email) DO NOTHING`, s.ID, email); err != nil {
			return err
		}
	}
	for _, name := range s.Participants {
		if _, err := tx.Exec(ctx,
			`INSERT INTO session_participants (session_id, name) VALUES ($1, $2)`, s.ID, name); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (p *PG) GetSession(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	var s domain.Session
	err := p.pool.QueryRow(ctx, `
		SELECT id, title, status, summary_style, send_policy, cancel_window_seconds,
		       COALESCE(error_message, ''), created_at, updated_at, started_at, ended_at
		FROM sessions WHERE id = $1`, id).Scan(
		&s.ID, &s.Title, &s.Status, &s.SummaryStyle, &s.SendPolicy, &s.CancelWindowSeconds,
		&s.ErrorMessage, &s.CreatedAt, &s.UpdatedAt, &s.StartedAt, &s.EndedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := p.pool.Query(ctx, `SELECT email FROM session_recipients WHERE session_id = $1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			rows.Close()
			return nil, err
		}
		s.Recipients = append(s.Recipients, e)
	}
	rows.Close()

	prows, err := p.pool.Query(ctx, `SELECT name FROM session_participants WHERE session_id = $1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	for prows.Next() {
		var n string
		if err := prows.Scan(&n); err != nil {
			prows.Close()
			return nil, err
		}
		s.Participants = append(s.Participants, n)
	}
	prows.Close()

	return &s, nil
}

func (p *PG) ListSessions(ctx context.Context, limit int) ([]domain.SessionListItem, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := p.pool.Query(ctx, `
		SELECT s.id, s.title, s.status, s.send_policy, s.created_at, s.ended_at,
		       (SELECT count(*) FROM session_recipients r WHERE r.session_id = s.id)
		FROM sessions s
		ORDER BY s.created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SessionListItem
	for rows.Next() {
		var it domain.SessionListItem
		if err := rows.Scan(&it.ID, &it.Title, &it.Status, &it.SendPolicy,
			&it.CreatedAt, &it.EndedAt, &it.RecipientCount); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (p *PG) UpdateSessionStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE sessions SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	return err
}

func (p *PG) SetSessionError(ctx context.Context, id uuid.UUID, msg string) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE sessions SET status = $2, error_message = $3, updated_at = now() WHERE id = $1`,
		id, domain.StatusFailed, msg)
	return err
}

func (p *PG) MarkStarted(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE sessions SET status = $2, started_at = $3, updated_at = now() WHERE id = $1`,
		id, domain.StatusRecording, at)
	return err
}

func (p *PG) MarkEnded(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE sessions SET status = $2, ended_at = $3, updated_at = now() WHERE id = $1`,
		id, domain.StatusProcessing, at)
	return err
}

// --- Audio chunks ---

func (p *PG) AddChunk(ctx context.Context, c *domain.AudioChunk) error {
	return p.pool.QueryRow(ctx, `
		INSERT INTO audio_chunks (session_id, seq, storage_path, size_bytes)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (session_id, seq) DO UPDATE SET storage_path = EXCLUDED.storage_path,
		                                            size_bytes = EXCLUDED.size_bytes
		RETURNING id, created_at`,
		c.SessionID, c.Seq, c.StoragePath, c.SizeBytes).Scan(&c.ID, &c.CreatedAt)
}

func (p *PG) ListChunks(ctx context.Context, sessionID uuid.UUID) ([]domain.AudioChunk, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, session_id, seq, storage_path, size_bytes, created_at
		FROM audio_chunks WHERE session_id = $1 ORDER BY seq`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AudioChunk
	for rows.Next() {
		var c domain.AudioChunk
		if err := rows.Scan(&c.ID, &c.SessionID, &c.Seq, &c.StoragePath, &c.SizeBytes, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// --- Transcripts ---

func (p *PG) CreateTranscript(ctx context.Context, t *domain.Transcript) error {
	return p.pool.QueryRow(ctx, `
		INSERT INTO transcripts (session_id, provider, language, text)
		VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		t.SessionID, t.Provider, t.Language, t.Text).Scan(&t.ID, &t.CreatedAt)
}

func (p *PG) GetTranscript(ctx context.Context, sessionID uuid.UUID) (*domain.Transcript, error) {
	var t domain.Transcript
	err := p.pool.QueryRow(ctx, `
		SELECT id, session_id, provider, language, text, created_at
		FROM transcripts WHERE session_id = $1 ORDER BY created_at DESC LIMIT 1`, sessionID).Scan(
		&t.ID, &t.SessionID, &t.Provider, &t.Language, &t.Text, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// --- Summaries ---

func (p *PG) CreateSummary(ctx context.Context, s *domain.Summary) error {
	cj, err := json.Marshal(s.Content)
	if err != nil {
		return err
	}
	return p.pool.QueryRow(ctx, `
		INSERT INTO summaries (session_id, style, content_json, content_text)
		VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		s.SessionID, s.Style, cj, s.ContentText).Scan(&s.ID, &s.CreatedAt)
}

func (p *PG) LatestSummary(ctx context.Context, sessionID uuid.UUID) (*domain.Summary, error) {
	var s domain.Summary
	var cj []byte
	err := p.pool.QueryRow(ctx, `
		SELECT id, session_id, style, content_json, content_text, created_at
		FROM summaries WHERE session_id = $1 ORDER BY created_at DESC LIMIT 1`, sessionID).Scan(
		&s.ID, &s.SessionID, &s.Style, &cj, &s.ContentText, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(cj, &s.Content); err != nil {
		return nil, err
	}
	return &s, nil
}

// --- Jobs ---

func (p *PG) CreateJob(ctx context.Context, sessionID uuid.UUID, typ string, runAt time.Time) (*domain.Job, error) {
	var j domain.Job
	err := p.pool.QueryRow(ctx, `
		INSERT INTO jobs (session_id, type, status, run_at)
		VALUES ($1, $2, 'pending', $3)
		RETURNING id, session_id, type, status, run_at, attempts, created_at, updated_at`,
		sessionID, typ, runAt).Scan(
		&j.ID, &j.SessionID, &j.Type, &j.Status, &j.RunAt, &j.Attempts, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// ClaimNextJob, çalışmaya hazır (run_at <= now) bir bekleyen işi atomik olarak alır.
// FOR UPDATE SKIP LOCKED ile çoklu worker güvenli.
func (p *PG) ClaimNextJob(ctx context.Context, now time.Time) (*domain.Job, error) {
	var j domain.Job
	err := p.pool.QueryRow(ctx, `
		UPDATE jobs SET status = 'running', attempts = attempts + 1, updated_at = now()
		WHERE id = (
			SELECT id FROM jobs
			WHERE status = 'pending' AND run_at <= $1
			ORDER BY run_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, session_id, type, status, run_at, attempts, created_at, updated_at`, now).Scan(
		&j.ID, &j.SessionID, &j.Type, &j.Status, &j.RunAt, &j.Attempts, &j.CreatedAt, &j.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (p *PG) CompleteJob(ctx context.Context, id int64) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE jobs SET status = 'done', updated_at = now() WHERE id = $1`, id)
	return err
}

func (p *PG) FailJob(ctx context.Context, id int64, errMsg string) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE jobs SET status = 'failed', last_error = $2, updated_at = now() WHERE id = $1`, id, errMsg)
	return err
}

// CancelPendingSend, bekleyen send işlerini iptal eder (cancel_window akışı).
func (p *PG) CancelPendingSend(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	tag, err := p.pool.Exec(ctx, `
		UPDATE jobs SET status = 'cancelled', updated_at = now()
		WHERE session_id = $1 AND type = 'send' AND status = 'pending'`, sessionID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// --- Email deliveries ---

func (p *PG) CreateDelivery(ctx context.Context, d *domain.EmailDelivery) error {
	return p.pool.QueryRow(ctx, `
		INSERT INTO email_deliveries (session_id, recipient, status)
		VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		d.SessionID, d.Recipient, d.Status).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (p *PG) UpdateDelivery(ctx context.Context, id int64, status, providerID, errMsg string) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE email_deliveries SET status = $2, provider_id = NULLIF($3, ''),
		       error_message = NULLIF($4, ''), updated_at = now() WHERE id = $1`,
		id, status, providerID, errMsg)
	return err
}

// --- Contacts ---

func (p *PG) ListContacts(ctx context.Context) ([]domain.Contact, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, email, name, created_at, last_used_at
		FROM contacts ORDER BY last_used_at DESC, email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Contact
	for rows.Next() {
		var c domain.Contact
		if err := rows.Scan(&c.ID, &c.Email, &c.Name, &c.CreatedAt, &c.LastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpsertContact, e-postayı ekler ya da varsa last_used_at'i günceller
// (ve boşsa ismi doldurur).
func (p *PG) UpsertContact(ctx context.Context, email, name string) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO contacts (email, name) VALUES ($1, $2)
		ON CONFLICT (email) DO UPDATE SET
			last_used_at = now(),
			name = CASE WHEN contacts.name = '' AND EXCLUDED.name <> '' THEN EXCLUDED.name ELSE contacts.name END`,
		email, name)
	return err
}

func (p *PG) UpdateContact(ctx context.Context, id int64, name string) error {
	_, err := p.pool.Exec(ctx, `UPDATE contacts SET name = $2 WHERE id = $1`, id, name)
	return err
}

func (p *PG) DeleteContact(ctx context.Context, id int64) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM contacts WHERE id = $1`, id)
	return err
}

// --- Groups (dağıtım listeleri) ---

func (p *PG) ListGroups(ctx context.Context) ([]domain.Group, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT g.id, g.name, g.created_at,
		       COALESCE(array_agg(m.email ORDER BY m.email) FILTER (WHERE m.email IS NOT NULL), '{}') AS emails
		FROM recipient_groups g
		LEFT JOIN recipient_group_members m ON m.group_id = g.id
		GROUP BY g.id
		ORDER BY g.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Group
	for rows.Next() {
		var g domain.Group
		if err := rows.Scan(&g.ID, &g.Name, &g.CreatedAt, &g.Emails); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (p *PG) CreateGroup(ctx context.Context, name string, emails []string) (*domain.Group, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var g domain.Group
	g.Name = name
	if err := tx.QueryRow(ctx,
		`INSERT INTO recipient_groups (name) VALUES ($1) RETURNING id, created_at`,
		name).Scan(&g.ID, &g.CreatedAt); err != nil {
		return nil, err
	}
	for _, e := range emails {
		if _, err := tx.Exec(ctx,
			`INSERT INTO recipient_group_members (group_id, email) VALUES ($1, $2)
			 ON CONFLICT (group_id, email) DO NOTHING`, g.ID, e); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	g.Emails = emails
	return &g, nil
}

func (p *PG) DeleteGroup(ctx context.Context, id int64) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM recipient_groups WHERE id = $1`, id)
	return err
}

func (p *PG) ListDeliveries(ctx context.Context, sessionID uuid.UUID) ([]domain.EmailDelivery, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, session_id, recipient, status, COALESCE(provider_id, ''),
		       COALESCE(error_message, ''), created_at, updated_at
		FROM email_deliveries WHERE session_id = $1 ORDER BY id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.EmailDelivery
	for rows.Next() {
		var d domain.EmailDelivery
		if err := rows.Scan(&d.ID, &d.SessionID, &d.Recipient, &d.Status,
			&d.ProviderID, &d.ErrorMessage, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
