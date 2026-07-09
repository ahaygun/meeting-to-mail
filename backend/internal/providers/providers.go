// Package providers, dış AI/mail servislerinin arayüzlerini ve stub implementasyonlarını tutar.
// Boru hattı önce stub'larla uçtan uca kurulur; sonra gerçek servisler takılır.
package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"meeting-to-mail/internal/domain"
)

// ASR, sesi metne döken servis.
type ASR interface {
	// Transcribe, birleştirilmiş ses dosyasını metne döker.
	Transcribe(ctx context.Context, audioPath, language string) (text string, provider string, err error)
}

// Summarizer, transkriptten yapılandırılmış özet üreten servis.
type Summarizer interface {
	Summarize(ctx context.Context, transcript, style string, participants []string) (domain.SummaryContent, error)
}

// Mailer, özeti alıcılara gönderen servis.
type Mailer interface {
	// Send, tek bir alıcıya gönderir; sağlayıcı mesaj ID'si döner.
	Send(ctx context.Context, to, subject, body string) (providerID string, err error)
}

// --- Stub implementasyonları ---

// StubASR, gerçek ASR yerine sabit/sahte bir transkript üretir.
type StubASR struct{}

func (StubASR) Transcribe(ctx context.Context, audioPath, language string) (string, string, error) {
	// Gerçek işlemeyi taklit etmek için kısa bir gecikme.
	select {
	case <-time.After(800 * time.Millisecond):
	case <-ctx.Done():
		return "", "stub", ctx.Err()
	}
	text := "Bu bir örnek (stub) transkripttir. Toplantıda proje planı gözden geçirildi, " +
		"kayıt boru hattının uçtan uca çalışması hedeflendi ve gerçek ASR servisi entegre " +
		"edilene kadar bu sahte metin kullanılacak. Ali sunumu hazırlayacak, Ayşe bütçeyi " +
		"gelecek hafta paylaşacak."
	return text, "stub", nil
}

// StubSummarizer, transkripti basitçe yapılandırılmış özete çevirir.
type StubSummarizer struct{}

func (StubSummarizer) Summarize(ctx context.Context, transcript, style string, participants []string) (domain.SummaryContent, error) {
	select {
	case <-time.After(600 * time.Millisecond):
	case <-ctx.Done():
		return domain.SummaryContent{}, ctx.Err()
	}
	return domain.SummaryContent{
		Headline: "Proje planı gözden geçirildi (örnek özet)",
		KeyPoints: []string{
			"Kayıt boru hattının uçtan uca çalışması hedeflendi.",
			"Gerçek ASR/LLM servisleri entegre edilene kadar stub kullanılacak.",
		},
		Decisions: []string{
			"Boru hattı önce stub'larla kurulacak, sonra gerçek servisler takılacak.",
		},
		ActionItems: []domain.ActionItem{
			{Task: "Sunumu hazırlamak", Owner: "Ali", Due: "gelecek hafta"},
			{Task: "Bütçeyi paylaşmak", Owner: "Ayşe", Due: "gelecek hafta"},
		},
	}, nil
}

// StubMailer, gerçekten mail göndermez; sadece log-benzeri bir ID döner.
type StubMailer struct{}

func (StubMailer) Send(ctx context.Context, to, subject, body string) (string, error) {
	select {
	case <-time.After(150 * time.Millisecond):
	case <-ctx.Done():
		return "", ctx.Err()
	}
	fmt.Printf("[stub-mailer] → %s | konu: %q | gövde: %d karakter\n", to, subject, len(body))
	return "stub-" + strings.ReplaceAll(to, "@", "-at-"), nil
}

// RenderText, yapılandırılmış özeti mail gövdesi (düz metin) olarak render eder.
func RenderText(title string, c domain.SummaryContent) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", title)
	fmt.Fprintf(&b, "%s\n\n", c.Headline)
	if len(c.KeyPoints) > 0 {
		b.WriteString("Ana Maddeler:\n")
		for _, k := range c.KeyPoints {
			fmt.Fprintf(&b, "  • %s\n", k)
		}
		b.WriteString("\n")
	}
	if len(c.Decisions) > 0 {
		b.WriteString("Kararlar:\n")
		for _, d := range c.Decisions {
			fmt.Fprintf(&b, "  • %s\n", d)
		}
		b.WriteString("\n")
	}
	if len(c.ActionItems) > 0 {
		b.WriteString("Aksiyon Maddeleri:\n")
		for _, a := range c.ActionItems {
			line := "  • " + a.Task
			if a.Owner != "" {
				line += " — " + a.Owner
			}
			if a.Due != "" {
				line += " (" + a.Due + ")"
			}
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}
