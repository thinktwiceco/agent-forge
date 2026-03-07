package fs

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// rgToGrepFlag maps rg long-form flags to their grep equivalents.
var rgToGrepFlag = map[string]string{
	"--ignore-case":        "-i",
	"--line-number":        "-n",
	"--files-with-matches": "-l",
	"--count":              "-c",
	"--word-regexp":        "-w",
	"--invert-match":       "-v",
	"--fixed-strings":      "-F",
	"--with-filename":      "-H",
	"--no-filename":        "-h",
	"--only-matching":      "-o",
	"--recursive":          "-r",
	"--no-messages":        "-s",
	"--quiet":              "-q",
	"--pcre2":              "-P",
}

// rgOnlyPrefixes are rg flags (matched by prefix) that have no grep equivalent.
// They are silently dropped during translation.
var rgOnlyPrefixes = []string{
	"--heading", "--no-heading",
	"--hidden", "--no-hidden",
	"--follow", "--no-follow",
	"--glob=", "-g",
	"--type=", "-t", "--type-not=", "-T",
	"--multiline", "-U",
	"--trim",
	"--vimgrep",
	"--json",
	"--stats",
	"--smart-case", "-S",
	"--no-ignore",
}

// translateRgFlagsToGrep converts ripgrep flags to grep-compatible flags.
//   - Long-form rg flags with a grep equivalent are translated.
//   - rg-only flags (no grep equivalent) are silently dropped.
//   - Everything else (short flags, --color=, --include=, -A, -B, -C, -m, -e, …)
//     is passed through unchanged.
func translateRgFlagsToGrep(flags []string) []string {
	out := make([]string, 0, len(flags))
	for _, f := range flags {
		// Translate known long-form flags.
		if mapped, ok := rgToGrepFlag[f]; ok {
			out = append(out, mapped)
			continue
		}

		// Drop rg-only flags (exact match or prefix match).
		drop := false
		for _, prefix := range rgOnlyPrefixes {
			if f == prefix || strings.HasPrefix(f, prefix) {
				drop = true
				break
			}
		}
		if drop {
			continue
		}

		// Pass everything else through unchanged.
		out = append(out, f)
	}
	return out
}

// hasRecursiveFlag reports whether the flag list already contains a recursive flag.
func hasRecursiveFlag(flags []string) bool {
	for _, f := range flags {
		if f == "-r" || f == "-R" || f == "--recursive" {
			return true
		}
	}
	return false
}

// grepFallback executes grep as a transparent substitute for ripgrep.
// It translates rg-compatible flags to grep flags, auto-adds -r for directories,
// and returns output in the same ripgrepResponse format so callers are unaffected.
func (fs *Fs) grepFallback(relativePath, validatedPath, pattern string, flags []string, offset, headLimit int) (string, error) {
	if _, err := exec.LookPath("grep"); err != nil {
		return "", fmt.Errorf("neither ripgrep (rg) nor grep is installed or in PATH")
	}

	grepFlags := translateRgFlagsToGrep(flags)

	// grep does not recurse automatically; add -r when the target is a directory.
	if info, err := os.Stat(validatedPath); err == nil && info.IsDir() {
		if !hasRecursiveFlag(grepFlags) {
			grepFlags = append([]string{"-r"}, grepFlags...)
		}
		grepFlags = append(grepFlags, "--exclude=.env", "--exclude=.env.*", "--exclude=*.env")
	}

	args := append(grepFlags, pattern, validatedPath)
	cmd := exec.Command("grep", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// grep exit code 1 = no matches (not an error).
	// Exit code 2+ = real error.
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				response := &ripgrepResponse{
					Pattern:      pattern,
					RelativePath: relativePath,
					AbsolutePath: validatedPath,
					Flags:        flags,
					TotalMatches: 0,
					MatchCount:   0,
					Offset:       offset,
					HeadLimit:    headLimit,
					Output:       "",
					Status:       "No matches found",
				}
				return response.String(), nil
			}
			return "", fmt.Errorf("grep failed with exit code %d: %s", exitErr.ExitCode(), stderr.String())
		}
		return "", fmt.Errorf("failed to execute grep: %w", err)
	}

	// Collect non-empty match lines.
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	totalMatches := len(lines)

	// Apply offset.
	if offset > totalMatches {
		offset = totalMatches
	}
	lines = lines[offset:]

	// Apply head_limit (default 500 to guard against context overflow).
	if headLimit == 0 {
		headLimit = 500
	}
	if len(lines) > headLimit {
		lines = lines[:headLimit]
	}

	matchCount := len(lines)
	response := &ripgrepResponse{
		Pattern:      pattern,
		RelativePath: relativePath,
		AbsolutePath: validatedPath,
		Flags:        flags,
		TotalMatches: totalMatches,
		MatchCount:   matchCount,
		Offset:       offset,
		HeadLimit:    headLimit,
		Output:       strings.Join(lines, "\n"),
		Status:       fmt.Sprintf("Showing %d of %d matches", matchCount, totalMatches),
	}

	return response.String(), nil
}
