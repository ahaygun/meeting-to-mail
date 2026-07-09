// Package config, ortam değişkenlerinden uygulama ayarlarını okur.
package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config, uygulama ayarları.
type Config struct {
	Addr        string // HTTP dinleme adresi, ör. ":8080"
	DatabaseURL string // Postgres DSN
	AudioDir    string // ses parçalarının yazılacağı kök dizin
	MailFrom    string // giden mail "from" adresi
	CORSOrigin  string // izin verilen frontend origin

	// Sağlayıcılar — anahtar varsa gerçek servis, yoksa stub kullanılır.
	// ASR (ses→metin) — öncelik: yerel whisper.cpp > OpenAI Whisper > stub
	WhisperBin   string // whisper.cpp CLI, ör. "whisper-cli" (yerel/offline)
	WhisperModel string // ggml model yolu; ayarlıysa yerel whisper.cpp kullanılır
	OpenAIKey    string // OpenAI Whisper API (ASR) için
	ASRModel     string // ör. "whisper-1"
	ASRLanguage  string // ör. "tr"

	// LLM (özet) — öncelik: yerel Ollama > Google Gemini > stub
	OllamaHost  string // ör. "http://localhost:11434"
	OllamaModel string // ayarlıysa yerel Ollama kullanılır, ör. "qwen2.5:7b"
	GoogleKey   string // Google Gemini (özet) için
	LLMModel    string // ör. "gemini-2.5-flash"

	ResendKey string // Resend (mail) için
}

// Load, .env dosyasını (varsa) yükler ve Config döner.
func Load() Config {
	_ = godotenv.Load() // .env yoksa sorun değil

	return Config{
		Addr:        getenv("ADDR", ":8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://toplanti:toplanti@localhost:5434/toplanti?sslmode=disable"),
		AudioDir:    getenv("AUDIO_DIR", "./data/audio"),
		MailFrom:    getenv("MAIL_FROM", "toplanti@example.com"),
		CORSOrigin:  getenv("CORS_ORIGIN", "http://localhost:5173"),

		WhisperBin:   getenv("WHISPER_BIN", "whisper-cli"),
		WhisperModel: os.Getenv("WHISPER_MODEL"),
		OpenAIKey:    os.Getenv("OPENAI_API_KEY"),
		ASRModel:     getenv("ASR_MODEL", "whisper-1"),
		ASRLanguage:  firstNonEmpty(os.Getenv("WHISPER_LANG"), getenv("ASR_LANGUAGE", "tr")),
		OllamaHost:   getenv("OLLAMA_HOST", "http://localhost:11434"),
		OllamaModel:  os.Getenv("OLLAMA_MODEL"),
		GoogleKey:    firstNonEmpty(os.Getenv("GOOGLE_API_KEY"), os.Getenv("GEMINI_API_KEY")),
		LLMModel:     getenv("LLM_MODEL", "gemini-2.5-flash"),
		ResendKey:    os.Getenv("RESEND_API_KEY"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// firstNonEmpty, ilk boş olmayan değeri döner.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
