package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type WebhookSender struct {
	client       *http.Client
	merchantURLs map[string]string
}

func NewWebhookSender() *WebhookSender {
	return &WebhookSender{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		merchantURLs: map[string]string{
			"merchant-1": "https://merchant1.requestcatcher.com/",
			"merchant-2": "https://merchant2.requestcatcher.com/",
		},
	}
}

func (s *WebhookSender) Send(payload *WebhookPayload) *WebhookResult {
	var lastResult *WebhookResult

	for attempt := 1; attempt <= MaxRetries; attempt++ {
		result := s.doSend(payload)
		if result.Success {
			result.Attempts = attempt
			return result
		}

		lastResult = result
		lastResult.Attempts = attempt

		slog.Warn("webhook send failed",
			"attempt", attempt,
			"status_code", lastResult.StatusCode,
			"error", lastResult.Error,
		)

		if lastResult.Permanent {
			return lastResult
		}

		if attempt < MaxRetries {
			time.Sleep(RetryDelay * time.Duration(1<<(attempt-1)))

		}

	}
	return lastResult
}

func (s *WebhookSender) doSend(payload *WebhookPayload) *WebhookResult {
	start := time.Now()
	url, ok := s.merchantURLs[payload.MerchantID]
	if !ok {
		return &WebhookResult{
			Success:   false,
			Error:     fmt.Errorf("no webhook URL configured for merchant: %s", payload.MerchantID),
			Duration:  time.Since(start),
			Permanent: true,
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return &WebhookResult{
			Success:   false,
			Error:     fmt.Errorf("failed to marshal payload for merchant %s: %w", payload.MerchantID, err),
			Duration:  time.Since(start),
			Permanent: true,
		}
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return &WebhookResult{
			Success:   false,
			Error:     fmt.Errorf("failed to create request for merchant %s: %w", payload.MerchantID, err),
			Duration:  time.Since(start),
			Permanent: true,
		}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return &WebhookResult{
			Success:   false,
			Error:     fmt.Errorf("failed to send webhook to merchant %s: %w", payload.MerchantID, err),
			Duration:  time.Since(start),
			Permanent: false,
		}
	}
	defer resp.Body.Close()

	return &WebhookResult{
		Success:    resp.StatusCode >= 200 && resp.StatusCode < 300,
		StatusCode: resp.StatusCode,
		Duration:   time.Since(start),
		Permanent:  false,
	}
}
