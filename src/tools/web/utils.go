package web

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	defaultTimeout     = 60 * time.Second
	defaultSettleDelay = 500 * time.Millisecond
)

// waitForPageReady waits for the page to be fully ready for content extraction.
// Phase 1: polls document.readyState until it equals 'complete', ensuring all
// scripts have executed (critical for JS-heavy SPAs).
// Phase 2: holds a short settle delay to allow async data fetches and re-renders
// triggered by those scripts to finish. Pass 0 to skip the settle delay.
func waitForPageReady(ctx context.Context, settleDelay time.Duration) error {
	if err := chromedp.Run(ctx,
		chromedp.Poll(`document.readyState === 'complete'`, nil,
			chromedp.WithPollingInterval(200*time.Millisecond),
		),
	); err != nil {
		return err
	}
	if settleDelay > 0 {
		_ = chromedp.Run(ctx, chromedp.Sleep(settleDelay))
	}
	return nil
}

// normalizeURL validates and normalizes a URL by adding scheme if missing.
// Returns the normalized URL or an error if the URL is invalid.
func normalizeURL(urlStr string) (string, error) {
	if urlStr == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}

	// Add scheme if missing
	normalizedURL := urlStr
	if !strings.HasPrefix(normalizedURL, "http://") && !strings.HasPrefix(normalizedURL, "https://") {
		normalizedURL = "https://" + normalizedURL
	}

	parsedURL, err := url.Parse(normalizedURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("URL must use http or https scheme")
	}

	if parsedURL.Host == "" {
		return "", fmt.Errorf("URL must have a valid host")
	}

	return normalizedURL, nil
}

// parseTimeout parses a timeout value from args and returns it as time.Duration.
// Supports float64, int, and int64 types.
// Returns an error if the value is invalid or <= 0.
func parseTimeout(args map[string]any, key string, defaultDuration time.Duration) (time.Duration, error) {
	ts, ok := args[key]
	if !ok {
		return defaultDuration, nil
	}

	var seconds float64
	switch v := ts.(type) {
	case float64:
		seconds = v
	case int:
		seconds = float64(v)
	case int64:
		seconds = float64(v)
	default:
		return 0, fmt.Errorf("%s parameter must be a number", key)
	}

	if seconds <= 0 {
		return 0, fmt.Errorf("%s must be greater than 0", key)
	}

	return time.Duration(seconds) * time.Second, nil
}
