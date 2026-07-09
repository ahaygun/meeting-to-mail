// Toplantı Kayıt & Özet Otomasyonu — HTTP sunucusu + arka plan worker.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"meeting-to-mail/internal/config"
	"meeting-to-mail/internal/events"
	"meeting-to-mail/internal/httpapi"
	"meeting-to-mail/internal/providers"
	"meeting-to-mail/internal/storage"
	"meeting-to-mail/internal/store"
	"meeting-to-mail/internal/worker"
)

//go:generate echo "no codegen"

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Store (Postgres).
	pg, err := store.NewPG(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("veritabanına bağlanılamadı: %v", err)
	}
	defer pg.Close()

	// Şema migration'ını uygula (idempotent).
	if err := applyMigrations(ctx, pg); err != nil {
		log.Fatalf("migration hatası: %v", err)
	}

	// Ses depolama (yerel disk).
	disk, err := storage.NewDisk(cfg.AudioDir)
	if err != nil {
		log.Fatalf("ses dizini oluşturulamadı: %v", err)
	}

	// Olay hub'ı (SSE).
	hub := events.New()

	// Sağlayıcılar — yapılandırma varsa gerçek servis, yoksa stub.
	// ASR önceliği: yerel whisper.cpp (offline) > OpenAI Whisper (API) > stub.
	var asr providers.ASR = providers.StubASR{}
	switch {
	case cfg.WhisperModel != "":
		asr = providers.NewWhisperCppASR(cfg.WhisperBin, cfg.WhisperModel)
		log.Printf("[providers] ASR: whisper.cpp yerel (%s)", cfg.WhisperModel)
	case cfg.OpenAIKey != "":
		asr = providers.NewWhisperASR(cfg.OpenAIKey, cfg.ASRModel)
		log.Printf("[providers] ASR: OpenAI Whisper API (%s)", cfg.ASRModel)
	default:
		log.Println("[providers] ASR: stub (WHISPER_MODEL / OPENAI_API_KEY yok)")
	}

	// LLM önceliği: yerel Ollama (offline) > Google Gemini (API) > stub.
	var sum providers.Summarizer = providers.StubSummarizer{}
	switch {
	case cfg.OllamaModel != "":
		sum = providers.NewOllamaSummarizer(cfg.OllamaHost, cfg.OllamaModel)
		log.Printf("[providers] LLM: Ollama yerel (%s @ %s)", cfg.OllamaModel, cfg.OllamaHost)
	case cfg.GoogleKey != "":
		sum = providers.NewGeminiSummarizer(cfg.GoogleKey, cfg.LLMModel)
		log.Printf("[providers] LLM: Gemini API (%s)", cfg.LLMModel)
	default:
		log.Println("[providers] LLM: stub (OLLAMA_MODEL / GOOGLE_API_KEY yok)")
	}

	var mailer providers.Mailer = providers.StubMailer{}
	if cfg.ResendKey != "" {
		mailer = providers.NewResendMailer(cfg.ResendKey, cfg.MailFrom)
		log.Printf("[providers] Mail: Resend (from %s)", cfg.MailFrom)
	} else {
		log.Println("[providers] Mail: stub (RESEND_API_KEY yok)")
	}

	// Worker'ı başlat.
	wrk := worker.New(pg, disk, asr, sum, mailer, hub, cfg.MailFrom, cfg.ASRLanguage)
	go wrk.Run(ctx)

	// HTTP sunucusu.
	srv := httpapi.NewServer(pg, disk, hub, cfg.CORSOrigin)
	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("[http] dinleniyor %s", cfg.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http sunucu hatası: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("kapatılıyor…")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("kapatma hatası: %v", err)
	}
}

// applyMigrations, db/migrations altındaki .sql dosyalarını sırayla çalıştırır.
// Basit ve idempotent (CREATE TABLE IF NOT EXISTS) — küçük bir migrator yeterli.
func applyMigrations(ctx context.Context, pg *store.PG) error {
	const dir = "db/migrations"
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("[migrate] %s okunamadı (%v) — atlanıyor", dir, err)
		return nil
	}
	for _, e := range entries {
		if e.IsDir() || len(e.Name()) < 4 || e.Name()[len(e.Name())-4:] != ".sql" {
			continue
		}
		sqlBytes, err := os.ReadFile(dir + "/" + e.Name())
		if err != nil {
			return err
		}
		if _, err := pg.Pool().Exec(ctx, string(sqlBytes)); err != nil {
			return err
		}
		log.Printf("[migrate] uygulandı: %s", e.Name())
	}
	return nil
}
