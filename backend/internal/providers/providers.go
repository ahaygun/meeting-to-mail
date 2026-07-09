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

// TranscriptCleaner, özetten ÖNCE transkriptteki bariz ASR (ses tanıma)
// hatalarını bağlamdan onaran opsiyonel yetenek. Bir Summarizer bunu
// implemente ederse worker özetten önce çağırır.
type TranscriptCleaner interface {
	CleanTranscript(ctx context.Context, transcript string) (string, error)
}

// genericOwnerPhrases, gerçek bir sahip/isim OLMAYAN, modelin uydurduğu
// jenerik ifadeler. Bunlar sahip alanından temizlenir.
var genericOwnerPhrases = []string{
	"gündemi", "ilgili kişi", "ilgili birim", "birisi", "bir kişi",
	"sorumlu kişi", "tutacak kişi", "görevli kişi", "yetkili kişi",
	"atanacak", "belirlenecek", "tüm", "herkes", "ekip", "takım",
	"n/a", "yok", "belirsiz", "-",
}

// dueGenericPhrases, gerçek bir tarih OLMAYAN belirsiz ifadeler.
var dueGenericPhrases = []string{
	"belirlenecek", "belirsiz", "yakında", "en kısa sürede", "ileride", "n/a", "yok", "-",
}

// SanitizeSummary, çıktıyı deterministik olarak temizler:
//   - modelin uydurduğu jenerik owner/due değerlerini boşaltır,
//   - istenmeyen kelimeleri (ör. "yaratmak") uygun karşılıklarıyla değiştirir.
//
// Model talimatı dinlemese bile çalışır.
func SanitizeSummary(c domain.SummaryContent) domain.SummaryContent {
	c.Headline = replaceWords(c.Headline)
	for i := range c.KeyPoints {
		c.KeyPoints[i] = replaceWords(c.KeyPoints[i])
	}
	for i := range c.Decisions {
		c.Decisions[i] = replaceWords(c.Decisions[i])
	}
	for i := range c.ActionItems {
		c.ActionItems[i].Task = replaceWords(c.ActionItems[i].Task)
		c.ActionItems[i].Owner = sanitizeField(replaceWords(c.ActionItems[i].Owner), genericOwnerPhrases, 30)
		c.ActionItems[i].Due = sanitizeField(c.ActionItems[i].Due, dueGenericPhrases, 40)
	}
	return c
}

// replaceWords, dini/kültürel hassasiyet gereği istenmeyen kelimeleri değiştirir.
// "yaratmak" fiili yerine "oluşturmak" — çekimli hâller dahil (ünlü uyumuna dikkat).
func replaceWords(s string) string {
	repl := strings.NewReplacer(
		"yaratıl", "oluşturul", // pasif: yaratılması → oluşturulması
		"Yaratıl", "Oluşturul",
		"yarat", "oluştur", // yaratma → oluşturma, yaratmak → oluşturmak
		"Yarat", "Oluştur",
	)
	return repl.Replace(s)
}

func sanitizeField(v string, generic []string, maxLen int) string {
	t := strings.TrimSpace(v)
	if t == "" {
		return ""
	}
	low := strings.ToLower(t)
	for _, g := range generic {
		if strings.Contains(low, g) {
			return ""
		}
	}
	if len([]rune(t)) > maxLen {
		return ""
	}
	return t
}

// NewCorrections, "yanlış=>doğru; yanlış2=>doğru2" biçimindeki düzeltme
// sözlüğünü bir Replacer'a çevirir. Bilinen alan/kurum terimlerinin ASR
// hatalarını deterministik olarak düzeltmek için (ör. "iyitim=>iyilik").
func NewCorrections(spec string) *strings.Replacer {
	var pairs []string
	for _, part := range strings.Split(spec, ";") {
		kv := strings.SplitN(part, "=>", 2)
		if len(kv) != 2 {
			continue
		}
		wrong := strings.TrimSpace(kv[0])
		right := strings.TrimSpace(kv[1])
		if wrong == "" {
			continue
		}
		pairs = append(pairs, wrong, right)
	}
	if len(pairs) == 0 {
		return nil
	}
	return strings.NewReplacer(pairs...)
}

// ApplyCorrectionsToSummary, düzeltme sözlüğünü özet metin alanlarına uygular.
func ApplyCorrectionsToSummary(r *strings.Replacer, c domain.SummaryContent) domain.SummaryContent {
	if r == nil {
		return c
	}
	c.Headline = r.Replace(c.Headline)
	for i := range c.KeyPoints {
		c.KeyPoints[i] = r.Replace(c.KeyPoints[i])
	}
	for i := range c.Decisions {
		c.Decisions[i] = r.Replace(c.Decisions[i])
	}
	for i := range c.ActionItems {
		c.ActionItems[i].Task = r.Replace(c.ActionItems[i].Task)
		c.ActionItems[i].Owner = r.Replace(c.ActionItems[i].Owner)
	}
	return c
}

// cleanSystemPrompt, transkript düzeltme geçişinin sistem yönergesi (Ollama + Gemini paylaşır).
const cleanSystemPrompt = "Verilen Türkçe toplantı transkripti otomatik ses tanımayla (ASR) " +
	"üretildi ve BARİZ HATALAR içerir: yanlış duyulmuş kelimeler, bozuk yazım. " +
	"Görevin yalnızca bu bariz hataları BAĞLAMDAN düzeltmek.\n" +
	"KURALLAR:\n" +
	"- İçerik EKLEME veya ÇIKARMA; anlamı ve konuşma sırasını koru.\n" +
	"- Cümleleri yeniden yazma, özetleme, yorumlama.\n" +
	"- Emin olmadığın kelimeye DOKUNMA.\n" +
	"- Yalnızca düzeltilmiş transkript METNİNİ döndür; açıklama/başlık/madde EKLEME."

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
