// Package domain, uygulamanın çekirdek tiplerini ve durum sabitlerini tutar.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Session durumları — durum makinesi (PROJE_PLANI md §4).
const (
	StatusConfiguring  = "configuring"
	StatusRecording    = "recording"
	StatusProcessing   = "processing"
	StatusTranscribing = "transcribing"
	StatusSummarizing  = "summarizing"
	StatusPendingSend  = "pending_send"
	StatusSending      = "sending"
	StatusSent         = "sent"
	StatusCancelled    = "cancelled"
	StatusFailed       = "failed"
)

// Gönderim politikaları.
const (
	SendImmediate    = "immediate"
	SendCancelWindow = "cancel_window"
)

// İş tipleri.
const (
	JobTranscribe = "transcribe"
	JobSummarize  = "summarize"
	JobSend       = "send"
)

// İş durumları.
const (
	JobPending   = "pending"
	JobRunning   = "running"
	JobDone      = "done"
	JobFailed    = "failed"
	JobCancelled = "cancelled"
)

// Session, kayıt öncesi konfig + durum.
type Session struct {
	ID                  uuid.UUID  `json:"id"`
	Title               string     `json:"title"`
	Status              string     `json:"status"`
	SummaryStyle        string     `json:"summary_style"`
	SendPolicy          string     `json:"send_policy"`
	CancelWindowSeconds int        `json:"cancel_window_seconds"`
	ErrorMessage        string     `json:"error_message,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	StartedAt           *time.Time `json:"started_at,omitempty"`
	EndedAt             *time.Time `json:"ended_at,omitempty"`

	Recipients   []string `json:"recipients,omitempty"`
	Participants []string `json:"participants,omitempty"`
}

// Contact, kayıtlı bir alıcı (kişi rehberi girdisi).
type Contact struct {
	ID         int64     `json:"id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
}

// SessionListItem, geçmiş oturum listesinde gösterilen özet satır.
type SessionListItem struct {
	ID             uuid.UUID  `json:"id"`
	Title          string     `json:"title"`
	Status         string     `json:"status"`
	SendPolicy     string     `json:"send_policy"`
	CreatedAt      time.Time  `json:"created_at"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	RecipientCount int        `json:"recipient_count"`
}

// AudioChunk, parça parça gelen sesin metaverisi.
type AudioChunk struct {
	ID          int64     `json:"id"`
	SessionID   uuid.UUID `json:"session_id"`
	Seq         int       `json:"seq"`
	StoragePath string    `json:"storage_path"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}

// Transcript, ASR çıktısı.
type Transcript struct {
	ID        int64     `json:"id"`
	SessionID uuid.UUID `json:"session_id"`
	Provider  string    `json:"provider"`
	Language  string    `json:"language"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// ActionItem, yapılandırılmış özetteki bir aksiyon maddesi.
type ActionItem struct {
	Task  string `json:"task"`
	Owner string `json:"owner,omitempty"`
	Due   string `json:"due,omitempty"`
}

// SummaryContent, LLM'in ürettiği yapılandırılmış özet.
type SummaryContent struct {
	Headline    string       `json:"headline"`
	KeyPoints   []string     `json:"key_points"`
	Decisions   []string     `json:"decisions"`
	ActionItems []ActionItem `json:"action_items"`
}

// Summary, bir özet satırı (çoklu olabilir).
type Summary struct {
	ID          int64          `json:"id"`
	SessionID   uuid.UUID      `json:"session_id"`
	Style       string         `json:"style"`
	Content     SummaryContent `json:"content"`
	ContentText string         `json:"content_text"`
	CreatedAt   time.Time      `json:"created_at"`
}

// Job, async iş kuyruğu satırı.
type Job struct {
	ID        int64     `json:"id"`
	SessionID uuid.UUID `json:"session_id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	RunAt     time.Time `json:"run_at"`
	Attempts  int       `json:"attempts"`
	LastError string    `json:"last_error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// EmailDelivery, alıcı bazında gönderim logu.
type EmailDelivery struct {
	ID           int64     `json:"id"`
	SessionID    uuid.UUID `json:"session_id"`
	Recipient    string    `json:"recipient"`
	Status       string    `json:"status"`
	ProviderID   string    `json:"provider_id,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
