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
// Türkçe için ggml-small model iyi bir denge. Yerel işleme olduğu için 25MB gibi
// bir limit yoktur; birleşik ses tek seferde işlenir.
type WhisperCppASR struct {
	Bin   string // "whisper-cli"
	Model string // ör. .../ggml-small.bin
}

// NewWhisperCppASR bir WhisperCppASR oluşturur.
func NewWhisperCppASR(bin, model string) *WhisperCppASR {
	if bin == "" {
		bin = "whisper-cli"
	}
	return &WhisperCppASR{Bin: bin, Model: model}
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

	cmd := exec.CommandContext(ctx, w.Bin,
		"-m", w.Model,
		"-f", wavPath,
		"-l", language,
		"-nt",   // zaman damgası yok
		"-otxt", // düz metin transkript üret
		"-of", outBase,
	)
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
