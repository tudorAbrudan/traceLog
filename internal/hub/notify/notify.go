package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/wneessen/go-mail"
)

type Channel struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Config string `json:"config"`
}

type EmailConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	To       string `json:"to"`
	UseTLS   bool   `json:"use_tls"`
}

type WebhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

type Manager struct {
	channels map[string]*Channel
}

func NewManager() *Manager {
	return &Manager{
		channels: make(map[string]*Channel),
	}
}

func (m *Manager) AddChannel(ch *Channel) {
	m.channels[ch.ID] = ch
}

func (m *Manager) RemoveChannel(id string) {
	delete(m.channels, id)
}

func (m *Manager) Send(ctx context.Context, channelID string, subject, body string) error {
	ch, ok := m.channels[channelID]
	if !ok {
		return fmt.Errorf("channel %s not found", channelID)
	}

	switch ch.Type {
	case "email":
		return m.sendEmail(ctx, ch, subject, body)
	case "webhook":
		return m.sendWebhook(ctx, ch, subject, body)
	default:
		return fmt.Errorf("unknown channel type: %s", ch.Type)
	}
}

func (m *Manager) sendEmail(_ context.Context, ch *Channel, subject, body string) error {
	var cfg EmailConfig
	if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
		return fmt.Errorf("invalid email config: %w", err)
	}

	msg := mail.NewMsg()
	if err := msg.From(cfg.From); err != nil {
		return err
	}
	if err := msg.To(cfg.To); err != nil {
		return err
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextPlain, body)

	opts := []mail.Option{
		mail.WithPort(cfg.Port),
		mail.WithUsername(cfg.Username),
		mail.WithPassword(cfg.Password),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
	}
	if cfg.UseTLS {
		opts = append(opts, mail.WithSSLPort(false))
	}

	client, err := mail.NewClient(cfg.Host, opts...)
	if err != nil {
		return fmt.Errorf("create mail client: %w", err)
	}

	return client.DialAndSend(msg)
}

func (m *Manager) sendWebhook(ctx context.Context, ch *Channel, subject, body string) error {
	var cfg WebhookConfig
	if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
		return fmt.Errorf("invalid webhook config: %w", err)
	}

	method := cfg.Method
	if method == "" {
		method = "POST"
	}

	payload, _ := json.Marshal(map[string]string{
		"subject": subject,
		"body":    body,
		"time":    time.Now().UTC().Format(time.RFC3339),
	})

	req, err := http.NewRequestWithContext(ctx, method, cfg.URL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}
