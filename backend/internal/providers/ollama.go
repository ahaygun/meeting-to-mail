package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"meeting-to-mail/internal/domain"
)

// OllamaSummarizer, yerel Ollama sunucusunu (localhost:11434) kullanan
// TAMAMEN YEREL (offline) özet sağlayıcısı — veri cihazdan çıkmaz, API anahtarı
// gerekmez. `format` alanına JSON şeması verilerek yapılandırılmış çıktı alınır.
type OllamaSummarizer struct {
	Host   string // ör. "http://localhost:11434"
	Model  string // ör. "qwen2.5:7b"
	client *http.Client
}

// NewOllamaSummarizer bir OllamaSummarizer oluşturur.
func NewOllamaSummarizer(host, model string) *OllamaSummarizer {
	if host == "" {
		host = "http://localhost:11434"
	}
	if model == "" {
		model = "qwen2.5:7b"
	}
	return &OllamaSummarizer{
		Host:   strings.TrimRight(host, "/"),
		Model:  model,
		client: &http.Client{Timeout: 5 * time.Minute},
	}
}

type ollamaChatReq struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Format   map[string]any  `json:"format,omitempty"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResp struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Error string `json:"error"`
}

// ollamaSchema, yapılandırılmış özetin JSON Schema karşılığı (Ollama structured outputs).
func ollamaSchema() map[string]any {
	str := map[string]any{"type": "string"}
	strArr := map[string]any{"type": "array", "items": str}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"headline":   str,
			"key_points": strArr,
			"decisions":  strArr,
			"action_items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"task":  str,
						"owner": str,
						"due":   str,
					},
					"required": []string{"task", "owner", "due"},
				},
			},
		},
		"required": []string{"headline", "key_points", "decisions", "action_items"},
	}
}

// Summarize, transkriptten yapılandırılmış özet üretir.
func (o *OllamaSummarizer) Summarize(ctx context.Context, transcript, style string, participants []string) (domain.SummaryContent, error) {
	var empty domain.SummaryContent

	system := "Sen bir toplantı asistanısın. Verilen Türkçe toplantı transkriptinden " +
		"yapılandırılmış bir özet çıkar: başlık (headline), ana maddeler (key_points), " +
		"kararlar (decisions), aksiyon maddeleri (action_items: task/owner/due). " +
		"Yanıtın TÜRKÇE olsun.\n" +
		"KURALLAR:\n" +
		"- owner (sahip): SADECE transkriptte bir işi belirli bir KİŞİYE/İSME açıkça atanmışsa doldur. " +
		"Belirsizse BOŞ bırak. 'gündeminde tutacak kişi', 'ilgili kişi', 'birisi' gibi genel ifadeleri sahip olarak YAZMA.\n" +
		"- due (tarih): sadece açıkça bir tarih/süre söylendiyse doldur, yoksa BOŞ bırak. Tarih UYDURMA.\n" +
		"- Transkriptte olmayan bilgi ekleme, kişi/kurum adı uydurma.\n" +
		"- ÖZ OL, TEKRARLAMA: key_points tartışılan ana konulardır (en fazla 6). " +
		"decisions yalnızca net alınan kararlardır. action_items yalnızca yapılacak somut işlerdir. " +
		"Aynı maddeyi birden fazla bölüme KOYMA (bir madde ya karar ya aksiyondur).\n" + styleGuide(style)

	var b strings.Builder
	if len(participants) > 0 {
		fmt.Fprintf(&b, "Katılımcılar: %s\n\n", strings.Join(participants, ", "))
	}
	b.WriteString("Transkript:\n")
	b.WriteString(transcript)

	reqBody := ollamaChatReq{
		Model: o.Model,
		Messages: []ollamaMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: b.String()},
		},
		Stream: false,
		Format: ollamaSchema(),
		// num_ctx: uzun toplantı transkriptlerinin sığması için bağlamı genişlet
		// (varsayılan 4096 uzun toplantılarda yetmeyebilir).
		Options: map[string]any{"temperature": 0.2, "num_ctx": 16384},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return empty, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.Host+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return empty, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return empty, fmt.Errorf("ollama'ya bağlanılamadı (%s): %w", o.Host, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var or ollamaChatResp
	if err := json.Unmarshal(raw, &or); err != nil {
		return empty, fmt.Errorf("ollama yanıtı ayrıştırılamadı (%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if resp.StatusCode != http.StatusOK || or.Error != "" {
		msg := or.Error
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		return empty, fmt.Errorf("ollama API %d: %s", resp.StatusCode, msg)
	}

	text := stripJSONFences(or.Message.Content)
	var content domain.SummaryContent
	if err := json.Unmarshal([]byte(text), &content); err != nil {
		return empty, fmt.Errorf("özet JSON'u ayrıştırılamadı: %w — gelen: %s", err, text)
	}
	return content, nil
}

// CleanTranscript, transkriptteki bariz ASR hatalarını bağlamdan onarır (düz metin döner).
func (o *OllamaSummarizer) CleanTranscript(ctx context.Context, transcript string) (string, error) {
	reqBody := ollamaChatReq{
		Model: o.Model,
		Messages: []ollamaMessage{
			{Role: "system", Content: cleanSystemPrompt},
			{Role: "user", Content: transcript},
		},
		Stream:  false,
		Options: map[string]any{"temperature": 0.1, "num_ctx": 16384},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return transcript, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.Host+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return transcript, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.client.Do(req)
	if err != nil {
		return transcript, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var or ollamaChatResp
	if err := json.Unmarshal(raw, &or); err != nil || resp.StatusCode != http.StatusOK || or.Error != "" {
		return transcript, fmt.Errorf("ollama temizleme hatası (%d)", resp.StatusCode)
	}
	out := strings.TrimSpace(or.Message.Content)
	if out == "" {
		return transcript, nil
	}
	return out, nil
}
