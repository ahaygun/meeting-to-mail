package providers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

// WhisperASR, OpenAI Whisper API'ını kullanan gerçek ASR sağlayıcısı.
// Uzun sesi ffmpeg ile ~20 dk'lık parçalara böler (her biri 25MB limitinin altında),
// her parçayı ayrı transkribe edip sonuçları birleştirir.
type WhisperASR struct {
	APIKey         string
	Model          string // "whisper-1"
	SegmentSeconds int    // parça süresi; 0 → 1200 (20 dk)
	client         *http.Client
}

// NewWhisperASR bir WhisperASR oluşturur.
func NewWhisperASR(apiKey, model string) *WhisperASR {
	if model == "" {
		model = "whisper-1"
	}
	return &WhisperASR{
		APIKey:         apiKey,
		Model:          model,
		SegmentSeconds: 1200,
		client:         &http.Client{Timeout: 10 * time.Minute},
	}
}

// Transcribe, birleşik ses dosyasını metne döker.
func (w *WhisperASR) Transcribe(ctx context.Context, audioPath, language string) (string, string, error) {
	segments, cleanup, err := prepareSegments(ctx, audioPath, w.SegmentSeconds)
	if err != nil {
		return "", "whisper", err
	}
	defer cleanup()

	var b strings.Builder
	for i, seg := range segments {
		text, err := w.transcribeSegment(ctx, seg, language)
		if err != nil {
			return "", "whisper", fmt.Errorf("parça %d transkript hatası: %w", i, err)
		}
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(strings.TrimSpace(text))
	}
	return b.String(), "whisper:" + w.Model, nil
}

// transcribeSegment, tek bir ses parçasını Whisper API'ına gönderir.
func (w *WhisperASR) transcribeSegment(ctx context.Context, path, language string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	fw, err := mw.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return "", err
	}
	_ = mw.WriteField("model", w.Model)
	if language != "" {
		_ = mw.WriteField("language", language)
	}
	_ = mw.WriteField("response_format", "text")
	if err := mw.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/audio/transcriptions", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+w.APIKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whisper API %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	// response_format=text → düz metin döner.
	return string(body), nil
}
