// Package worker, jobs tablosundan işleri çekip boru hattını yürüten
// arka plan goroutine'ini içerir: transcribe → summarize → send.
package worker

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"

	"meeting-to-mail/internal/domain"
	"meeting-to-mail/internal/events"
	"meeting-to-mail/internal/providers"
	"meeting-to-mail/internal/storage"
	"meeting-to-mail/internal/store"
)

// Worker, boru hattını yürüten bileşen.
type Worker struct {
	st       store.Store
	disk     *storage.Disk
	asr      providers.ASR
	sum      providers.Summarizer
	mailer   providers.Mailer
	hub      *events.Hub
	mailFrom string
	asrLang  string
	poll     time.Duration
}

// New bir Worker oluşturur.
func New(st store.Store, disk *storage.Disk, asr providers.ASR, sum providers.Summarizer,
	mailer providers.Mailer, hub *events.Hub, mailFrom, asrLang string) *Worker {
	if asrLang == "" {
		asrLang = "tr"
	}
	return &Worker{
		st: st, disk: disk, asr: asr, sum: sum, mailer: mailer,
		hub: hub, mailFrom: mailFrom, asrLang: asrLang, poll: 400 * time.Millisecond,
	}
}

// Run, ctx iptal edilene kadar işleri çeker. Kendi goroutine'inde çağrılmalı.
func (w *Worker) Run(ctx context.Context) {
	log.Println("[worker] başladı")
	ticker := time.NewTicker(w.poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("[worker] durdu")
			return
		case <-ticker.C:
			w.drain(ctx)
		}
	}
}

// drain, hazır işleri arka arkaya işler (kuyruk boşalana dek).
func (w *Worker) drain(ctx context.Context) {
	for {
		job, err := w.st.ClaimNextJob(ctx, time.Now())
		if errors.Is(err, store.ErrNotFound) {
			return
		}
		if err != nil {
			log.Printf("[worker] iş alınamadı: %v", err)
			return
		}
		w.process(ctx, job)
	}
}

func (w *Worker) process(ctx context.Context, job *domain.Job) {
	var err error
	switch job.Type {
	case domain.JobTranscribe:
		err = w.doTranscribe(ctx, job.SessionID)
	case domain.JobSummarize:
		err = w.doSummarize(ctx, job.SessionID)
	case domain.JobSend:
		err = w.doSend(ctx, job.SessionID)
	default:
		err = errors.New("bilinmeyen iş tipi: " + job.Type)
	}

	if err != nil {
		log.Printf("[worker] iş #%d (%s) hata: %v", job.ID, job.Type, err)
		_ = w.st.FailJob(ctx, job.ID, err.Error())
		_ = w.st.SetSessionError(ctx, job.SessionID, err.Error())
		w.hub.Publish(job.SessionID, events.Event{Status: domain.StatusFailed, Step: job.Type, Message: err.Error()})
		return
	}
	_ = w.st.CompleteJob(ctx, job.ID)
}

func (w *Worker) doTranscribe(ctx context.Context, sessionID uuid.UUID) error {
	_ = w.st.UpdateSessionStatus(ctx, sessionID, domain.StatusTranscribing)
	w.hub.Publish(sessionID, events.Event{Status: domain.StatusTranscribing, Step: domain.JobTranscribe, Message: "Ses metne dökülüyor…"})

	chunks, err := w.st.ListChunks(ctx, sessionID)
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(chunks))
	for _, c := range chunks {
		paths = append(paths, c.StoragePath)
	}
	combined, err := w.disk.Concatenate(sessionID, paths)
	if err != nil {
		return err
	}
	text, provider, err := w.asr.Transcribe(ctx, combined, w.asrLang)
	if err != nil {
		return err
	}
	if err := w.st.CreateTranscript(ctx, &domain.Transcript{
		SessionID: sessionID, Provider: provider, Language: w.asrLang, Text: text,
	}); err != nil {
		return err
	}
	// Sonraki adımı kuyruğa al.
	_, err = w.st.CreateJob(ctx, sessionID, domain.JobSummarize, time.Now())
	return err
}

func (w *Worker) doSummarize(ctx context.Context, sessionID uuid.UUID) error {
	_ = w.st.UpdateSessionStatus(ctx, sessionID, domain.StatusSummarizing)
	w.hub.Publish(sessionID, events.Event{Status: domain.StatusSummarizing, Step: domain.JobSummarize, Message: "Özet çıkarılıyor…"})

	sess, err := w.st.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	tr, err := w.st.GetTranscript(ctx, sessionID)
	if err != nil {
		return err
	}
	content, err := w.sum.Summarize(ctx, tr.Text, sess.SummaryStyle, sess.Participants)
	if err != nil {
		return err
	}
	text := providers.RenderText(sess.Title, content)
	if err := w.st.CreateSummary(ctx, &domain.Summary{
		SessionID: sessionID, Style: sess.SummaryStyle, Content: content, ContentText: text,
	}); err != nil {
		return err
	}

	// Gönderim politikasına göre send işini planla.
	switch sess.SendPolicy {
	case domain.SendCancelWindow:
		runAt := time.Now().Add(time.Duration(sess.CancelWindowSeconds) * time.Second)
		_ = w.st.UpdateSessionStatus(ctx, sessionID, domain.StatusPendingSend)
		w.hub.Publish(sessionID, events.Event{
			Status: domain.StatusPendingSend, Step: domain.JobSummarize,
			Message: "Özet hazır — gönderim iptal penceresinde bekliyor.",
		})
		_, err = w.st.CreateJob(ctx, sessionID, domain.JobSend, runAt)
	default: // immediate
		_, err = w.st.CreateJob(ctx, sessionID, domain.JobSend, time.Now())
	}
	return err
}

func (w *Worker) doSend(ctx context.Context, sessionID uuid.UUID) error {
	_ = w.st.UpdateSessionStatus(ctx, sessionID, domain.StatusSending)
	w.hub.Publish(sessionID, events.Event{Status: domain.StatusSending, Step: domain.JobSend, Message: "E-postalar gönderiliyor…"})

	sess, err := w.st.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	sum, err := w.st.LatestSummary(ctx, sessionID)
	if err != nil {
		return err
	}

	for _, to := range sess.Recipients {
		d := &domain.EmailDelivery{SessionID: sessionID, Recipient: to, Status: "pending"}
		if err := w.st.CreateDelivery(ctx, d); err != nil {
			return err
		}
		providerID, sendErr := w.mailer.Send(ctx, to, sess.Title, sum.ContentText)
		if sendErr != nil {
			_ = w.st.UpdateDelivery(ctx, d.ID, "failed", "", sendErr.Error())
			continue
		}
		_ = w.st.UpdateDelivery(ctx, d.ID, "sent", providerID, "")
	}

	_ = w.st.UpdateSessionStatus(ctx, sessionID, domain.StatusSent)
	w.hub.Publish(sessionID, events.Event{Status: domain.StatusSent, Step: domain.JobSend, Message: "Özet gönderildi."})
	return nil
}
