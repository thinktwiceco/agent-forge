package heartbeat

import (
	"fmt"
	"sync"
)

// HeartbeatManager maintains a named set of instructions that are injected
// into the heartbeat prompt. Each instruction is stored under a unique title.
type HeartbeatManager struct {
	mu           sync.Mutex
	instructions map[string]string // title → body
}

// NewHeartbeatManager returns an empty manager.
func NewHeartbeatManager() *HeartbeatManager {
	return &HeartbeatManager{
		instructions: make(map[string]string),
	}
}

// AddInstruction inserts or replaces an instruction under the given title.
// The title must be non-empty.
func (m *HeartbeatManager) AddInstruction(title, instruction string) error {
	if title == "" {
		return fmt.Errorf("heartbeat manager: title must not be empty")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instructions[title] = instruction
	return nil
}

// RemoveInstruction deletes the instruction with the given title.
// Returns an error when the title does not exist.
func (m *HeartbeatManager) RemoveInstruction(title string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.instructions[title]; !ok {
		return fmt.Errorf("heartbeat manager: instruction %q not found", title)
	}
	delete(m.instructions, title)
	return nil
}

// ListInstructions returns all instruction titles in insertion-stable order
// (sorted alphabetically for determinism).
func (m *HeartbeatManager) ListInstructions() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	titles := make([]string, 0, len(m.instructions))
	for t := range m.instructions {
		titles = append(titles, t)
	}
	sortStrings(titles)
	return titles
}

// renderInstructions formats all instructions as markdown sections
// (## Title\nbody) for inclusion in the heartbeat prompt.
func (m *HeartbeatManager) renderInstructions() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.instructions) == 0 {
		return ""
	}
	titles := make([]string, 0, len(m.instructions))
	for t := range m.instructions {
		titles = append(titles, t)
	}
	sortStrings(titles)

	var out string
	for _, t := range titles {
		out += "## " + t + "\n" + m.instructions[t] + "\n\n"
	}
	return out
}

// sortStrings sorts a string slice in place (stdlib sort is not imported to
// avoid an extra dependency — use a simple insertion sort for small slices).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}
