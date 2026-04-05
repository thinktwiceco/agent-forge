package heartbeat

import (
	"fmt"
	"time"
)

// isWithinActiveHours reports whether now falls inside the configured time window.
// Returns true (allow) on any parse error so a misconfigured range never silently
// suppresses all heartbeats.
func isWithinActiveHours(h *HoursRange, now time.Time) bool {
	loc := time.Local
	if h.Timezone != "" {
		if l, err := time.LoadLocation(h.Timezone); err == nil {
			loc = l
		}
	}
	t := now.In(loc)

	start, err := parseHHMM(h.Start)
	if err != nil {
		return true
	}
	end, err := parseHHMM(h.End)
	if err != nil {
		return true
	}

	cur := t.Hour()*60 + t.Minute()
	return cur >= start && cur < end
}

// parseHHMM parses "HH:MM" or the special value "24:00" into minutes since midnight.
func parseHHMM(s string) (int, error) {
	if s == "24:00" {
		return 24 * 60, nil
	}
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return 0, fmt.Errorf("heartbeat: invalid time %q", s)
	}
	return h*60 + m, nil
}
