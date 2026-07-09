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
)

// ResendMailer, Resend API üzerinden gerçek e-posta gönderir.
type ResendMailer struct {
	APIKey string
	From   string
	client *http.Client
}

// NewResendMailer bir ResendMailer oluşturur.
func NewResendMailer(apiKey, from string) *ResendMailer {
	return &ResendMailer{
		APIKey: apiKey,
		From:   from,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type resendReq struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
}

type resendResp struct {
	ID    string `json:"id"`
	Name  string `json:"name"`    // hata adı (varsa)
	Error string `json:"message"` // hata mesajı (varsa)
}

// Send, tek bir alıcıya e-posta gönderir; Resend mesaj ID'si döner.
func (m *ResendMailer) Send(ctx context.Context, to, subject, body string) (string, error) {
	payload, err := json.Marshal(resendReq{
		From: m.From, To: []string{to}, Subject: subject, Text: body,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.resend.com/emails", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+m.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var rr resendResp
	_ = json.Unmarshal(raw, &rr)
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if rr.Error != "" {
			msg = rr.Error
		}
		return "", fmt.Errorf("resend API %d: %s", resp.StatusCode, msg)
	}
	return rr.ID, nil
}
