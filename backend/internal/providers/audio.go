package providers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

// ffmpegAvailable, ffmpeg'in PATH'te olup olmadığını kontrol eder.
func ffmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// transcodeToWAV, herhangi bir ses dosyasını whisper.cpp'nin beklediği
// 16kHz mono WAV formatına çevirir. Yerel işleme olduğu için bölme gerekmez.
// Dönen cleanup fonksiyonu çağıran tarafından çağrılmalıdır.
func transcodeToWAV(ctx context.Context, inputPath string) (wavPath string, cleanup func(), err error) {
	if !ffmpegAvailable() {
		return "", func() {}, fmt.Errorf("ffmpeg bulunamadı (kurulum: brew install ffmpeg)")
	}
	tmpDir, err := os.MkdirTemp("", "toplanti-wav-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup = func() { _ = os.RemoveAll(tmpDir) }

	wavPath = filepath.Join(tmpDir, "audio.wav")
	cmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error",
		"-y", "-i", inputPath, "-ac", "1", "-ar", "16000", wavPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("ffmpeg WAV dönüştürme hatası: %v: %s", err, string(out))
	}
	return wavPath, cleanup, nil
}

// prepareSegments, birleşik sesi ASR'a uygun parçalara hazırlar:
//  1. 16kHz mono MP3'e (64kbps) yeniden kodlar — Whisper zaten 16kHz kullanır,
//     bu adım dosyayı küçültür ve formatı normalleştirir.
//  2. ~20 dakikalık (segmentSeconds) parçalara böler — 64kbps'te her parça
//     ~9-10MB olur, Whisper'ın 25MB limitinin rahat altında.
//
// Dönen dizin ve dosyalar çağıran tarafından temizlenmelidir (cleanup fonksiyonu).
func prepareSegments(ctx context.Context, inputPath string, segmentSeconds int) (segments []string, cleanup func(), err error) {
	if !ffmpegAvailable() {
		return nil, func() {}, fmt.Errorf("ffmpeg bulunamadı: uzun/geçerli ses bölme için gereklidir (kurulum: brew install ffmpeg)")
	}

	tmpDir, err := os.MkdirTemp("", "toplanti-asr-*")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup = func() { _ = os.RemoveAll(tmpDir) }

	// 1) Normalize + küçült.
	normalized := filepath.Join(tmpDir, "audio.mp3")
	cmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error",
		"-y", "-i", inputPath,
		"-ac", "1", "-ar", "16000", "-b:a", "64k",
		normalized)
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("ffmpeg dönüştürme hatası: %v: %s", err, string(out))
	}

	// 2) Süreye göre parçalara böl. Tek parça olsa bile segment muxer çalışır.
	if segmentSeconds <= 0 {
		segmentSeconds = 1200 // 20 dk
	}
	pattern := filepath.Join(tmpDir, "seg_%03d.mp3")
	segCmd := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error",
		"-y", "-i", normalized,
		"-f", "segment", "-segment_time", fmt.Sprintf("%d", segmentSeconds),
		"-c", "copy", pattern)
	if out, err := segCmd.CombinedOutput(); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("ffmpeg bölme hatası: %v: %s", err, string(out))
	}

	// Üretilen parçaları sırayla topla.
	matches, err := filepath.Glob(filepath.Join(tmpDir, "seg_*.mp3"))
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if len(matches) == 0 {
		// Segment üretilmediyse (çok kısa ses) normalize edilmiş dosyayı kullan.
		matches = []string{normalized}
	}
	sort.Strings(matches)
	return matches, cleanup, nil
}
