// Package storage, ses parçalarını yerel diske yazar ve birleştirir.
// MinIO/S3'e geçmeye hazır olacak şekilde basit bir arayüz sunar.
package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Disk, ses parçalarını yerel diskte saklayan storage.
type Disk struct {
	root string
}

// NewDisk verilen kök dizini oluşturur ve bir Disk döner.
func NewDisk(root string) (*Disk, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Disk{root: root}, nil
}

func (d *Disk) sessionDir(sessionID uuid.UUID) string {
	return filepath.Join(d.root, sessionID.String())
}

// SaveChunk bir parçayı diske yazar ve göreli depolama yolunu + byte sayısını döner.
func (d *Disk) SaveChunk(sessionID uuid.UUID, seq int, r io.Reader) (path string, size int64, err error) {
	dir := d.sessionDir(sessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, err
	}
	name := fmt.Sprintf("chunk_%06d.webm", seq)
	full := filepath.Join(dir, name)
	f, err := os.Create(full)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	size, err = io.Copy(f, r)
	if err != nil {
		return "", 0, err
	}
	// DB'de göreli yol saklıyoruz (taşınabilirlik için).
	rel := filepath.Join(sessionID.String(), name)
	return rel, size, nil
}

// Concatenate parçaları sırayla tek bir dosyada birleştirir ve tam yolu döner.
// Not: WebM/Opus parçalarının ham birleştirilmesi çoğu ASR için yeterlidir;
// gerekirse ileride ffmpeg ile yeniden paketleme eklenebilir.
func (d *Disk) Concatenate(sessionID uuid.UUID, chunkPaths []string) (string, error) {
	dir := d.sessionDir(sessionID)
	outPath := filepath.Join(dir, "combined.webm")
	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	for _, rel := range chunkPaths {
		full := filepath.Join(d.root, rel)
		in, err := os.Open(full)
		if err != nil {
			return "", fmt.Errorf("parça açılamadı %s: %w", rel, err)
		}
		if _, err := io.Copy(out, in); err != nil {
			in.Close()
			return "", err
		}
		in.Close()
	}
	return outPath, nil
}
