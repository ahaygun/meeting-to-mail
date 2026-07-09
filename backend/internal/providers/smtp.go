package providers

import (
	"context"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SMTPMailer, kurum-içi / yerel bir SMTP sunucusu üzerinden e-posta gönderir.
// Bulut SaaS'a (Resend) alternatif "yerel" ayak: veri kurumun kendi mail
// altyapısından geçer. Kullanıcı+parola boşsa kimlik doğrulamasız gönderir
// (ör. yerel geliştirmede Mailpit).
type SMTPMailer struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// NewSMTPMailer bir SMTPMailer oluşturur.
func NewSMTPMailer(host, port, username, password, from string) *SMTPMailer {
	return &SMTPMailer{Host: host, Port: port, Username: username, Password: password, From: from}
}

// Send, tek bir alıcıya e-posta gönderir; providerID olarak Message-ID döner.
// smtp.SendMail bağlam-farkında olmadığından iptali onurlandırmak için
// gorutinde çalıştırıp ctx.Done() ile yarıştırırız.
func (m *SMTPMailer) Send(ctx context.Context, to, subject, body string) (string, error) {
	addr := net.JoinHostPort(m.Host, m.Port)

	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}

	msgID := fmt.Sprintf("<%d@%s>", time.Now().UnixNano(), m.Host)
	msg := renderRFC822(m.From, to, subject, body, msgID)

	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, m.From, []string{to}, []byte(msg))
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("smtp gönderim (%s): %w", addr, err)
		}
		return msgID, nil
	}
}

// renderRFC822, UTF-8 gövdeli basit bir düz-metin e-posta kurar.
// Türkçe karakterli konu MIME (Q) ile kodlanır; gövde satır sonları CRLF'e çevrilir.
func renderRFC822(from, to, subject, body, msgID string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&b, "Message-ID: %s\r\n", msgID)
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))
	return b.String()
}
