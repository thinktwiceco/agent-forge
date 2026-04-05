package heartbeat

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// HeartbeatConfig holds all configuration for the heartbeat plugin.
type HeartbeatConfig struct {
	Every       string      `yaml:"every"`
	Prompt      string      `yaml:"prompt"`
	AckMaxChars int         `yaml:"ack_max_chars"`
	ActiveHours *HoursRange `yaml:"active_hours"`
}

// DefaultConfig returns the same defaults as the heartbeat registry factory and YAML when agent.heartbeat is omitted.
func DefaultConfig() HeartbeatConfig {
	return HeartbeatConfig{
		Every:       "30m",
		AckMaxChars: 300,
	}
}

// MergeConfig applies YAML-supplied overrides from agent.heartbeat into defaults.
// from is nil or the unmarshaled pointer when the heartbeat block is absent or present.
//
// Rules:
//   - Every: empty or omitted means use default "30m"; any non-empty string (including "0m", "0") is used as-is.
//   - Prompt: empty means use the built-in default prompt in resolvePrompt; non-empty overrides.
//   - AckMaxChars: 0 means use plugin default (300); non-zero overrides.
//   - ActiveHours: nil means no time window; non-nil sets active hours.
func MergeConfig(from *HeartbeatConfig) HeartbeatConfig {
	out := DefaultConfig()
	if from == nil {
		return out
	}
	if from.Every != "" {
		out.Every = from.Every
	}
	if from.Prompt != "" {
		out.Prompt = from.Prompt
	}
	if from.AckMaxChars != 0 {
		out.AckMaxChars = from.AckMaxChars
	}
	if from.ActiveHours != nil {
		out.ActiveHours = from.ActiveHours
	}
	return out
}

// HoursRange defines an active time window within which heartbeats may fire.
type HoursRange struct {
	Start    string `yaml:"start"`    // "HH:MM" inclusive
	End      string `yaml:"end"`      // "HH:MM" exclusive; "24:00" allowed
	Timezone string `yaml:"timezone"` // IANA or empty for host TZ
}

// parseInterval accepts a bare integer (treated as minutes) or a Go duration
// string (e.g. "30m", "1h"). Returns 0 for "0m" or "0" (disabled).
func parseInterval(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	if n, err := strconv.Atoi(s); err == nil {
		return time.Duration(n) * time.Minute, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("heartbeat: invalid interval %q: %w", s, err)
	}
	return d, nil
}
