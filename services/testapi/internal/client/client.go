package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	testapiconfig "avito-kitchen/services/testapi/internal/config"
	"avito-kitchen/services/testapi/internal/stock"
)

const (
	requestTimeout = 10 * time.Second

	apiKeyHeader = "X-API-Key"

	registerPlacePath = "/api/v1/b2b/places"
	menuPathFormat    = "/api/v1/b2b/places/%s/menu"
	deliveredFormat   = "/api/v1/b2b/orders/%s/delivered"
	trackingFormat    = "/api/v1/b2b/orders/%s/tracking"
	menuItemFormat    = "/api/v1/b2b/places/%s/menu/%s"
)

type Client struct {
	baseURL string
	client  *http.Client
}

func New(conf testapiconfig.Config) *Client {
	return &Client{
		baseURL: conf.KitchenBaseURL,
		client:  &http.Client{Timeout: requestTimeout},
	}
}

type Credentials struct {
	mu      sync.RWMutex
	placeID string
	apiKey  string
}

func NewCredentials() *Credentials {
	return &Credentials{}
}

func (c *Credentials) Set(placeID, apiKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.placeID = placeID
	c.apiKey = apiKey
}

func (c *Credentials) Get() (string, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.placeID, c.apiKey
}

func (c *Client) RegisterPlace(ctx context.Context, place RegisterPlaceRequest) (*RegisterPlaceResponse, error) {
	var resp RegisterPlaceResponse
	if err := c.do(ctx, http.MethodPost, registerPlacePath, "", place, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) AddMenu(ctx context.Context, placeID, apiKey string, items []stock.Item) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf(menuPathFormat, placeID), apiKey, toMenuRequest(items), nil)
}

func (c *Client) MarkDelivered(ctx context.Context, orderID, apiKey string) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf(deliveredFormat, orderID), apiKey, nil, nil)
}

func (c *Client) Restock(ctx context.Context, placeID, externalID, apiKey string, delta int32) error {
	return c.do(ctx, http.MethodPatch, fmt.Sprintf(menuItemFormat, placeID, externalID), apiKey,
		StockDeltaRequest{StockDelta: delta}, nil)
}

func (c *Client) ReportTracking(ctx context.Context, orderID, apiKey string, tracking TrackingRequest) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf(trackingFormat, orderID), apiKey, tracking, nil)
}

func (c *Client) do(ctx context.Context, method, path, apiKey string, body, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if apiKey != "" {
		req.Header.Set(apiKeyHeader, apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		details, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s %s: unexpected status %d: %s", method, path, resp.StatusCode, details)
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%s %s: decode response: %w", method, path, err)
	}

	return nil
}
