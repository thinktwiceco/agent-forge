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
	defaultTimeout       = 60 * time.Second
	defaultSettleDelay   = 500 * time.Millisecond
	networkIdleTimeout   = 10 * time.Second
	networkIdlePollEvery = 50 * time.Millisecond
)

// waitForPageReady waits for the page to be fully ready for content extraction.
// Phase 1: inject fetch/XHR counter, then poll document.readyState === 'complete'.
// Phase 2 (skipped if settleDelay == 0): wait until in-flight fetch/XHR count stays
// at 0 for settleDelay (network-idle threshold). Outer cap: networkIdleTimeout.
// settleDelay 0 skips the network-idle phase (backward compatible with callers that opt out).
func waitForPageReady(ctx context.Context, settleDelay time.Duration) error {
	if err := chromedp.Run(ctx, chromedp.Evaluate(getScript("network_idle"), nil)); err != nil {
		return err
	}
	if err := chromedp.Run(ctx,
		chromedp.Poll(`document.readyState === 'complete'`, nil,
			chromedp.WithPollingInterval(200*time.Millisecond),
		),
	); err != nil {
		return err
	}
	if settleDelay <= 0 {
		return nil
	}
	return waitForNetworkIdle(ctx, settleDelay)
}

// waitForNetworkIdle polls window.__agentPending until it remains 0 for threshold,
// or until networkIdleTimeout elapses. On timeout, returns nil (best effort) so pages
// with perpetual background polling do not fail navigate/get_content.
func waitForNetworkIdle(ctx context.Context, threshold time.Duration) error {
	deadline := time.Now().Add(networkIdleTimeout)
	var stableStart time.Time
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return nil
		}
		var pending float64
		if err := chromedp.Run(ctx, chromedp.Evaluate(`Number(window.__agentPending||0)`, &pending)); err != nil {
			return err
		}
		if pending == 0 {
			if stableStart.IsZero() {
				stableStart = time.Now()
			}
			if time.Since(stableStart) >= threshold {
				return nil
			}
		} else {
			stableStart = time.Time{}
		}
		if err := chromedp.Run(ctx, chromedp.Sleep(networkIdlePollEvery)); err != nil {
			return err
		}
	}
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
