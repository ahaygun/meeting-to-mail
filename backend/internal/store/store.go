// Package store, kalıcılık arayüzünü ve Postgres implementasyonunu tutar.
// Arayüz sayesinde ileride başka bir backend'e (ör. bellek içi test store) geçmek kolaydır.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"meeting-to-mail/internal/domain"
)

// ErrNotFound, aranan kayıt bulunamadığında döner.
var ErrNotFound = errors.New("kayıt bulunamadı")

// Store, uygulamanın ihtiyaç duyduğu tüm kalıcılık işlemleri.
type Store interface {
	// Sessions
	CreateSession(ctx context.Context, s *domain.Session) error
	GetSession(ctx context.Context, id uuid.UUID) (*domain.Session, error)
	ListSessions(ctx context.Context, limit int) ([]domain.SessionListItem, error)
	UpdateSessionStatus(ctx context.Context, id uuid.UUID, status string) error
	SetSessionError(ctx context.Context, id uuid.UUID, msg string) error
	MarkStarted(ctx context.Context, id uuid.UUID, at time.Time) error
	MarkEnded(ctx context.Context, id uuid.UUID, at time.Time) error

	// Audio chunks
	AddChunk(ctx context.Context, c *domain.AudioChunk) error
	ListChunks(ctx context.Context, sessionID uuid.UUID) ([]domain.AudioChunk, error)

	// Transcripts
	CreateTranscript(ctx context.Context, t *domain.Transcript) error
	GetTranscript(ctx context.Context, sessionID uuid.UUID) (*domain.Transcript, error)

	// Summaries
	CreateSummary(ctx context.Context, s *domain.Summary) error
	LatestSummary(ctx context.Context, sessionID uuid.UUID) (*domain.Summary, error)

	// Jobs
	CreateJob(ctx context.Context, sessionID uuid.UUID, typ string, runAt time.Time) (*domain.Job, error)
	ClaimNextJob(ctx context.Context, now time.Time) (*domain.Job, error)
	CompleteJob(ctx context.Context, id int64) error
	FailJob(ctx context.Context, id int64, errMsg string) error
	CancelPendingSend(ctx context.Context, sessionID uuid.UUID) (int64, error)

	// Email deliveries
	CreateDelivery(ctx context.Context, d *domain.EmailDelivery) error
	UpdateDelivery(ctx context.Context, id int64, status, providerID, errMsg string) error
	ListDeliveries(ctx context.Context, sessionID uuid.UUID) ([]domain.EmailDelivery, error)

	// Contacts (kayıtlı alıcılar)
	ListContacts(ctx context.Context) ([]domain.Contact, error)
	UpsertContact(ctx context.Context, email, name string) error
	UpdateContact(ctx context.Context, id int64, name string) error
	DeleteContact(ctx context.Context, id int64) error

	// Groups (dağıtım listeleri)
	ListGroups(ctx context.Context) ([]domain.Group, error)
	CreateGroup(ctx context.Context, name string, emails []string) (*domain.Group, error)
	DeleteGroup(ctx context.Context, id int64) error
}
