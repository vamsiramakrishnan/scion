// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hub

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// WebhookConfig holds configuration for a webhook endpoint.
type WebhookConfig struct {
	// URL is the endpoint to POST events to.
	URL string `json:"url" yaml:"url"`
	// Secret is used to generate HMAC-SHA256 signatures for verification.
	// If empty, no signature header is sent.
	Secret string `json:"secret,omitempty" yaml:"secret,omitempty"`
	// Events is the list of event subjects to subscribe to.
	// Uses NATS-style patterns: "grove.>", "agent.*.status", etc.
	// Empty means all events.
	Events []string `json:"events,omitempty" yaml:"events,omitempty"`
	// Enabled controls whether this webhook is active.
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// WebhookPayload is the JSON body sent to webhook endpoints.
type WebhookPayload struct {
	ID        string          `json:"id"`        // Unique event ID
	Timestamp string          `json:"timestamp"` // ISO 8601
	Subject   string          `json:"subject"`   // Event subject (e.g., "agent.abc.status")
	Data      json.RawMessage `json:"data"`      // Event-specific data
}

// WebhookDispatcher sends events to configured webhook endpoints.
type WebhookDispatcher struct {
	mu       sync.RWMutex
	webhooks []WebhookConfig
	client   *http.Client
	stopCh   chan struct{}
	wg       sync.WaitGroup

	// Metrics
	deliverySuccesses atomic.Int64
	deliveryFailures  atomic.Int64
}

// NewWebhookDispatcher creates a new webhook dispatcher.
func NewWebhookDispatcher() *WebhookDispatcher {
	return &WebhookDispatcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		stopCh: make(chan struct{}),
	}
}

// Configure sets the webhook endpoints.
func (wd *WebhookDispatcher) Configure(webhooks []WebhookConfig) {
	wd.mu.Lock()
	defer wd.mu.Unlock()
	wd.webhooks = webhooks
}

// Start begins listening to events from the publisher and dispatching to webhooks.
func (wd *WebhookDispatcher) Start(publisher *ChannelEventPublisher) {
	wd.mu.RLock()
	if len(wd.webhooks) == 0 {
		wd.mu.RUnlock()
		return
	}
	wd.mu.RUnlock()

	// Subscribe to all events
	ch, unsub := publisher.Subscribe(">")

	wd.wg.Add(1)
	go func() {
		defer wd.wg.Done()
		defer unsub()

		for {
			select {
			case evt, ok := <-ch:
				if !ok {
					return
				}
				wd.dispatch(evt)
			case <-wd.stopCh:
				return
			}
		}
	}()

	slog.Info("Webhook dispatcher started", "webhook_count", len(wd.webhooks))
}

// Stop signals the dispatcher to stop and waits for completion.
func (wd *WebhookDispatcher) Stop() {
	close(wd.stopCh)
	wd.wg.Wait()
}

// DeliverySuccesses returns the total successful deliveries.
func (wd *WebhookDispatcher) DeliverySuccesses() int64 {
	return wd.deliverySuccesses.Load()
}

// DeliveryFailures returns the total failed deliveries.
func (wd *WebhookDispatcher) DeliveryFailures() int64 {
	return wd.deliveryFailures.Load()
}

// dispatch sends an event to all matching webhook endpoints.
func (wd *WebhookDispatcher) dispatch(evt Event) {
	payload := WebhookPayload{
		ID:        fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Subject:   evt.Subject,
		Data:      json.RawMessage(evt.Data),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshal webhook payload", "error", err)
		return
	}

	wd.mu.RLock()
	webhooks := wd.webhooks
	wd.mu.RUnlock()

	for _, wh := range webhooks {
		if !wh.Enabled {
			continue
		}
		if len(wh.Events) > 0 && !matchesAnyPattern(wh.Events, evt.Subject) {
			continue
		}
		go wd.deliverWithRetry(wh, body)
	}
}

// deliverWithRetry attempts to deliver a webhook with exponential backoff.
func (wd *WebhookDispatcher) deliverWithRetry(wh WebhookConfig, body []byte) {
	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-time.After(backoff):
			case <-wd.stopCh:
				return
			}
		}

		if err := wd.deliver(wh, body); err != nil {
			if attempt == maxRetries {
				wd.deliveryFailures.Add(1)
				slog.Warn("Webhook delivery failed after retries",
					"url", wh.URL,
					"attempts", maxRetries+1,
					"error", err,
				)
			}
			continue
		}

		wd.deliverySuccesses.Add(1)
		return
	}
}

// deliver sends a single webhook HTTP POST.
func (wd *WebhookDispatcher) deliver(wh WebhookConfig, body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Scion-Webhook/1.0")
	req.Header.Set("X-Scion-Event-ID", "") // Populated from payload

	// Sign the payload with HMAC-SHA256 if a secret is configured
	if wh.Secret != "" {
		mac := hmac.New(sha256.New, []byte(wh.Secret))
		mac.Write(body)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Scion-Signature", "sha256="+signature)
	}

	resp, err := wd.client.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", wh.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("POST %s returned %d", wh.URL, resp.StatusCode)
}

// matchesAnyPattern checks if a subject matches any of the given patterns.
func matchesAnyPattern(patterns []string, subject string) bool {
	for _, p := range patterns {
		if subjectMatchesPattern(p, subject) {
			return true
		}
	}
	return false
}
