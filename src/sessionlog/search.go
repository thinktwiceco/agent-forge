package sessionlog

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

const defaultSearchHeadLimit = 50

// MatchLine is one regex match in a session log file.
type MatchLine struct {
	LineNumber int
	Text       string
}

// SearchResult holds paginated regex matches from a session log file.
type SearchResult struct {
	Pattern      string
	LogPath      string
	TotalMatches int
	MatchCount   int
	Offset       int
	HeadLimit    int
	Matches      []MatchLine
}

// SearchFile scans path line-by-line for regex matches without loading the whole file.
func SearchFile(path, pattern string, offset, headLimit int) (*SearchResult, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session log not found: %s", path)
		}
		return nil, fmt.Errorf("open session log: %w", err)
	}
	defer f.Close()

	if headLimit <= 0 {
		headLimit = defaultSearchHeadLimit
	}
	if offset < 0 {
		offset = 0
	}

	var all []MatchLine
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if re.MatchString(line) {
			all = append(all, MatchLine{LineNumber: lineNumber, Text: line})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read session log: %w", err)
	}

	total := len(all)
	if offset > total {
		offset = total
	}
	window := all[offset:]
	if len(window) > headLimit {
		window = window[:headLimit]
	}

	return &SearchResult{
		Pattern:      pattern,
		LogPath:      path,
		TotalMatches: total,
		MatchCount:   len(window),
		Offset:       offset,
		HeadLimit:    headLimit,
		Matches:      window,
	}, nil
}

func (r *SearchResult) String() string {
	status := "No matches found"
	if r.TotalMatches > 0 {
		status = fmt.Sprintf("Showing %d of %d matches", r.MatchCount, r.TotalMatches)
	}

	out := fmt.Sprintf(`Session Log Search
Pattern: %s
File: %s
Status: %s`, r.Pattern, r.LogPath, status)

	if len(r.Matches) > 0 {
		out += "\n\nMatches:\n---"
		for _, m := range r.Matches {
			out += fmt.Sprintf("\n%d: %s", m.LineNumber, m.Text)
		}
		out += "\n---"
	}

	remaining := r.TotalMatches - r.Offset - r.MatchCount
	if remaining > 0 {
		nextOffset := r.Offset + r.MatchCount
		out += fmt.Sprintf("\n... and %d more matches. Use offset=%d (and head_limit) to view more.", remaining, nextOffset)
	}

	return out
}
