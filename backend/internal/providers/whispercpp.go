package providers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WhisperCppASR, whisper.cpp CLI (`whisper-cli`) ile TAMAMEN YEREL (offline)
// ses→metin yapar — hiçbir ses cihazdan dışarı çıkmaz, API anahtarı gerekmez.
// Doğruluk için: beam search, sıcaklık 0, non-speech token bastırma, alan
// sözlüğü ipucu (initial prompt) ve opsiyonel VAD (sessizlik halüsinasyonunu keser).
type WhisperCppASR struct {
	Bin      string // "whisper-cli"
	Model    string // ör. .../ggml-large-v3.bin
	Prompt   string // alan sözlüğü ipucu (doğru yazımlara yönlendirir)
	VADModel string // ayarlıysa VAD etkinleşir (sessizliği kırpar)
}

// NewWhisperCppASR bir WhisperCppASR oluşturur.
func NewWhisperCppASR(bin, model, prompt, vadModel string) *WhisperCppASR {
	if bin == "" {
		bin = "whisper-cli"
	}
	return &WhisperCppASR{Bin: bin, Model: model, Prompt: prompt, VADModel: vadModel}
}

// Transcribe, birleşik ses dosyasını yerel whisper.cpp ile metne döker.
// whisper.cpp 16kHz mono WAV bekler; ffmpeg ile önce dönüştürülür.
func (w *WhisperCppASR) Transcribe(ctx context.Context, audioPath, language string) (string, string, error) {
	if language == "" {
		language = "tr"
	}

	wavPath, cleanupWav, err := transcodeToWAV(ctx, audioPath)
	if err != nil {
		return "", "whisper.cpp", err
	}
	defer cleanupWav()

	tmpDir, err := os.MkdirTemp("", "whisper-out-")
	if err != nil {
		return "", "whisper.cpp", err
	}
	defer os.RemoveAll(tmpDir)
	outBase := filepath.Join(tmpDir, "out")

	args := []string{
		"-m", w.Model,
		"-f", wavPath,
		"-l", language,
		"-bs", "5", // beam search — daha isabetli
		"-bo", "5", // best-of adayları
		"-tp", "0", // sıcaklık 0 (deterministik)
		"-sns",  // non-speech token'ları bastır (halüsinasyon azaltır)
		"-nt",   // zaman damgası yok
		"-otxt", // düz metin transkript üret
		"-of", outBase,
	}
	if w.Prompt != "" {
		args = append(args, "--prompt", w.Prompt)
	}
	if w.VADModel != "" {
		args = append(args, "--vad", "--vad-model", w.VADModel)
	}
	cmd := exec.CommandContext(ctx, w.Bin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", "whisper.cpp", fmt.Errorf("whisper.cpp hatası: %s: %w", strings.TrimSpace(string(out)), err)
	}

	data, err := os.ReadFile(outBase + ".txt")
	if err != nil {
		return "", "whisper.cpp", fmt.Errorf("transkript okunamadı: %w", err)
	}
	model := filepath.Base(w.Model)
	return strings.TrimSpace(string(data)), "whisper.cpp:" + model, nil
}
