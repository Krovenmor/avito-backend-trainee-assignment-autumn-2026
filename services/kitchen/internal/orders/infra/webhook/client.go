package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"avito-kitchen/services/kitchen/internal/orders/domain"

	"go.uber.org/zap"
)

const (
	requestTimeout  = 10 * time.Second
	maxResponseBody = 64 << 10

	decisionAccepted = "accepted"
	decisionRejected = "rejected"
)

type Client struct {
	client *http.Client
	log    *zap.Logger
}

func NewClient(log *zap.Logger) *Client {
	return &Client{
		client: &http.Client{Timeout: requestTimeout},
		log:    log,
	}
}

func (c *Client) CallOnce(ctx context.Context, webhookURL string, order domain.Order) (domain.Decision, bool) {
	body, err := json.Marshal(toOrderPayload(order))
	if err != nil {
		c.log.Error("marshal webhook payload", zap.String("order_id", order.ID), zap.Error(err))
		return domain.Decision{}, false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		c.log.Error("build webhook request", zap.String("order_id", order.ID), zap.Error(err))
		return domain.Decision{}, false
	}
	req.Header.Set("Content-Type", "application/json")

	c.log.Debug("calling place webhook",
		zap.String("order_id", order.ID),
		zap.String("webhook_url", webhookURL),
	)

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("webhook call failed", zap.String("order_id", order.ID), zap.Error(err))
		return domain.Decision{}, false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		c.log.Warn("webhook returned unexpected status",
			zap.String("order_id", order.ID), zap.Int("status", resp.StatusCode))
		return domain.Decision{}, false
	}

	var payload decisionPayload
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBody)).Decode(&payload); err != nil {
		c.log.Warn("decode webhook response", zap.String("order_id", order.ID), zap.Error(err))
		return domain.Decision{}, false
	}

	decision, ok := toDecision(payload)
	if !ok {
		c.log.Warn("webhook returned unknown decision",
			zap.String("order_id", order.ID), zap.String("decision", payload.Decision))
	}

	return decision, ok
}
