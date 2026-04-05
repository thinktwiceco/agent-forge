package brain

import (
	"strings"
	"time"
)

// PluginConfig holds YAML-driven options for the brain plugin (dreaming scheduler).
type PluginConfig struct {
	// Dream is "on" or "off" (case-insensitive; also true/false, yes/no, 1/0).
	// Empty defaults to on.
	Dream string `yaml:"dream,omitempty"`
	// DreamTime is local wall-clock time when RunPending runs, e.g. "02:00" or "14:30:00".
	// Empty defaults to "02:00".
	DreamTime string `yaml:"dreamTime,omitempty"`
}

// DefaultPluginConfig returns defaults: dreaming on at 02:00 local.
func DefaultPluginConfig() PluginConfig {
	return PluginConfig{Dream: "on", DreamTime: "02:00"}
}

// MergePluginConfig overlays YAML onto defaults.
func MergePluginConfig(in *PluginConfig) PluginConfig {
	out := DefaultPluginConfig()
	if in == nil {
		return out
	}
	if s := strings.TrimSpace(in.Dream); s != "" {
		out.Dream = s
	}
	if s := strings.TrimSpace(in.DreamTime); s != "" {
		out.DreamTime = s
	}
	return out
}

// DreamingEnabled reports whether scheduled dreaming should run (daily dreamTime).
// The dream tool ignores this and always runs when invoked.
func (c PluginConfig) DreamingEnabled() bool {
	s := strings.ToLower(strings.TrimSpace(c.Dream))
	switch s {
	case "", "on", "true", "yes", "1":
		return true
	case "off", "false", "no", "0":
		return false
	default:
		return true
	}
}

// parseDreamClock returns hour, minute, second for today's schedule in loc.
func (c PluginConfig) parseDreamClock(loc *time.Location) (hour, min, sec int, ok bool) {
	s := strings.TrimSpace(c.DreamTime)
	if s == "" {
		s = "02:00"
	}
	layouts := []string{"15:04:05", "15:04"}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, s, loc)
		if err == nil {
			return t.Hour(), t.Minute(), t.Second(), true
		}
	}
	return 0, 0, 0, false
}

// nextDreamRun returns the next local time strictly after `now` matching DreamTime.
func (c PluginConfig) nextDreamRun(now time.Time) time.Time {
	loc := now.Location()
	h, m, s, ok := c.parseDreamClock(loc)
	if !ok {
		h, m, s = 2, 0, 0
	}
	target := time.Date(now.Year(), now.Month(), now.Day(), h, m, s, 0, loc)
	if !target.After(now) {
		target = target.Add(24 * time.Hour)
	}
	return target
}
