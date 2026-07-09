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

// GeminiSummarizer, Google Gemini (Generative Language) API'ını kullanan özet sağlayıcısı.
// responseSchema ile yapılandırılmış JSON çıktısı garanti edilir.
type GeminiSummarizer struct {
	APIKey string
	Model  string // ör. "gemini-2.5-flash"
	client *http.Client
}

// styleGuide, özet stiline göre yönerge döner.
func styleGuide(style string) string {
	switch style {
	case "full_minutes":
		return "Tam tutanak: tartışılan her konuyu ayrıntılı, sırayla özetle."
	case "short":
		return "Kısa özet: yalnızca en kritik 2-3 nokta, olabildiğince öz."
	default: // decisions_actions
		return "Kararlar ve aksiyonlara odaklan: alınan kararları ve kim-ne-ne zaman yapacak aksiyonları net çıkar."
	}
}

// stripJSONFences, modelin ürettiği metinden olası ```json ... ``` bloğunu temizler.
func stripJSONFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		s = s[start : end+1]
	}
	return s
}

// NewGeminiSummarizer bir GeminiSummarizer oluşturur.
func NewGeminiSummarizer(apiKey, model string) *GeminiSummarizer {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &GeminiSummarizer{
		APIKey: apiKey,
		Model:  model,
		client: &http.Client{Timeout: 3 * time.Minute},
	}
}

type geminiReq struct {
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  geminiGenConfig `json:"generationConfig"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenConfig struct {
	ResponseMIMEType string         `json:"responseMimeType,omitempty"`
	ResponseSchema   map[string]any `json:"responseSchema,omitempty"`
}

type geminiResp struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// summarySchema, yapılandırılmış özetin Gemini responseSchema karşılığı.
func summarySchema() map[string]any {
	str := map[string]any{"type": "STRING"}
	return map[string]any{
		"type": "OBJECT",
		"properties": map[string]any{
			"headline":   str,
			"key_points": map[string]any{"type": "ARRAY", "items": str},
			"decisions":  map[string]any{"type": "ARRAY", "items": str},
			"action_items": map[string]any{
				"type": "ARRAY",
				"items": map[string]any{
					"type": "OBJECT",
					"properties": map[string]any{
						"task":  str,
						"owner": str,
						"due":   str,
					},
					"required": []string{"task"},
				},
			},
		},
		"required": []string{"headline", "key_points", "decisions", "action_items"},
	}
}

// Summarize, transkriptten yapılandırılmış özet üretir.
func (g *GeminiSummarizer) Summarize(ctx context.Context, transcript, style string, participants []string) (domain.SummaryContent, error) {
	var empty domain.SummaryContent

	system := "Sen bir toplantı asistanısın. Verilen Türkçe toplantı transkriptinden " +
		"yapılandırılmış bir özet çıkar: başlık (headline), ana maddeler (key_points), " +
		"kararlar (decisions), aksiyon maddeleri (action_items: task/owner/due).\n" +
		"KURALLAR:\n" +
		"- owner (sahip): SADECE bir iş belirli bir KİŞİYE/İSME açıkça atanmışsa doldur; " +
		"belirsizse BOŞ bırak. 'gündeminde tutacak kişi', 'ilgili kişi' gibi genel ifadeleri sahip olarak YAZMA.\n" +
		"- due (tarih): açıkça söylendiyse doldur, yoksa BOŞ bırak. UYDURMA.\n" +
		"- Transkriptte olmayan bilgi/isim ekleme.\n" +
		"- ÖZ OL, TEKRARLAMA: key_points en fazla 6 ana konu; decisions yalnızca net kararlar; " +
		"action_items yalnızca somut işler. Aynı maddeyi birden fazla bölüme KOYMA.\n" + styleGuide(style)

	var b strings.Builder
	if len(participants) > 0 {
		fmt.Fprintf(&b, "Katılımcılar: %s\n\n", strings.Join(participants, ", "))
	}
	b.WriteString("Transkript:\n")
	b.WriteString(transcript)

	reqBody := geminiReq{
		SystemInstruction: &geminiContent{Parts: []geminiPart{{Text: system}}},
		Contents:          []geminiContent{{Role: "user", Parts: []geminiPart{{Text: b.String()}}}},
		GenerationConfig: geminiGenConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema:   summarySchema(),
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return empty, err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", g.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return empty, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.APIKey)

	resp, err := g.client.Do(req)
	if err != nil {
		return empty, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var gr geminiResp
	if err := json.Unmarshal(raw, &gr); err != nil {
		return empty, fmt.Errorf("gemini yanıtı ayrıştırılamadı (%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if resp.StatusCode != http.StatusOK || gr.Error != nil {
		msg := strings.TrimSpace(string(raw))
		if gr.Error != nil {
			msg = gr.Error.Message
		}
		return empty, fmt.Errorf("gemini API %d: %s", resp.StatusCode, msg)
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return empty, fmt.Errorf("gemini boş yanıt döndü: %s", strings.TrimSpace(string(raw)))
	}

	text := stripJSONFences(gr.Candidates[0].Content.Parts[0].Text)
	var content domain.SummaryContent
	if err := json.Unmarshal([]byte(text), &content); err != nil {
		return empty, fmt.Errorf("özet JSON'u ayrıştırılamadı: %w — gelen: %s", err, text)
	}
	return content, nil
}

// CleanTranscript, transkriptteki bariz ASR hatalarını bağlamdan onarır (düz metin döner).
func (g *GeminiSummarizer) CleanTranscript(ctx context.Context, transcript string) (string, error) {
	reqBody := geminiReq{
		SystemInstruction: &geminiContent{Parts: []geminiPart{{Text: cleanSystemPrompt}}},
		Contents:          []geminiContent{{Role: "user", Parts: []geminiPart{{Text: transcript}}}},
		GenerationConfig:  geminiGenConfig{}, // düz metin
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return transcript, err
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", g.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return transcript, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.APIKey)
	resp, err := g.client.Do(req)
	if err != nil {
		return transcript, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var gr geminiResp
	if err := json.Unmarshal(raw, &gr); err != nil || resp.StatusCode != http.StatusOK || gr.Error != nil {
		return transcript, fmt.Errorf("gemini temizleme hatası (%d)", resp.StatusCode)
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return transcript, nil
	}
	out := strings.TrimSpace(gr.Candidates[0].Content.Parts[0].Text)
	if out == "" {
		return transcript, nil
	}
	return out, nil
}
