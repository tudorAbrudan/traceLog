package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/tudorAbrudan/tracelog/internal/hub/notify"
)

// ErrNotificationChannelNotFound is returned when UPDATE affects no row.
var ErrNotificationChannelNotFound = errors.New("notification channel not found")

func (s *Store) ListNotificationChannels(ctx context.Context) ([]notify.Channel, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, type, config FROM notification_channels ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []notify.Channel
	for rows.Next() {
		var ch notify.Channel
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config); err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, nil
}

func (s *Store) CreateNotificationChannel(ctx context.Context, ch *notify.Channel) error {
	if ch.ID == "" {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("generate notification channel id: %w", err)
		}
		ch.ID = hex.EncodeToString(b)
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO notification_channels (id, name, type, config) VALUES (?, ?, ?, ?)`,
		ch.ID, ch.Name, ch.Type, ch.Config,
	)
	return err
}

// UpdateNotificationChannel persists name, type, and config for an existing channel (id unchanged).
func (s *Store) UpdateNotificationChannel(ctx context.Context, ch *notify.Channel) error {
	if ch.ID == "" {
		return fmt.Errorf("channel id required")
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE notification_channels SET name = ?, type = ?, config = ? WHERE id = ?`,
		ch.Name, ch.Type, ch.Config, ch.ID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotificationChannelNotFound
	}
	return nil
}

func (s *Store) DeleteNotificationChannel(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM notification_channels WHERE id = ?`, id)
	return err
}
