package heartbeat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thinktwiceco/agent-forge/src/queue"
)

// ─── isEffectivelyEmpty ───────────────────────────────────────────────────────

func TestIsEffectivelyEmpty_HeadersOnly(t *testing.T) {
	content := "# My Checklist\n## Sub-heading\n"
	if !isEffectivelyEmpty(content) {
		t.Error("headers-only content should be effectively empty")
	}
}

func TestIsEffectivelyEmpty_EmptyBullets(t *testing.T) {
	content := "# Tasks\n- [ ]\n* \n+ \n"
	if !isEffectivelyEmpty(content) {
		t.Error("empty-bullet content should be effectively empty")
	}
}

func TestIsEffectivelyEmpty_RealContent(t *testing.T) {
	content := "# Tasks\n- Check Gmail\n"
	if isEffectivelyEmpty(content) {
		t.Error("content with real task should not be effectively empty")
	}
}

func TestIsEffectivelyEmpty_BlankFile(t *testing.T) {
	if !isEffectivelyEmpty("") {
		t.Error("blank file should be effectively empty")
	}
}

// ─── parseInterval ────────────────────────────────────────────────────────────

func TestParseInterval_GoString(t *testing.T) {
	d, err := parseInterval("30m")
	if err != nil || d != 30*time.Minute {
		t.Errorf("expected 30m, got %v err=%v", d, err)
	}
}

func TestParseInterval_OneHour(t *testing.T) {
	d, err := parseInterval("1h")
	if err != nil || d != time.Hour {
		t.Errorf("expected 1h, got %v err=%v", d, err)
	}
}

func TestParseInterval_BareInt(t *testing.T) {
	d, err := parseInterval("30")
	if err != nil || d != 30*time.Minute {
		t.Errorf("expected 30m, got %v err=%v", d, err)
	}
}

func TestParseInterval_Zero(t *testing.T) {
	d, err := parseInterval("0m")
	if err != nil || d != 0 {
		t.Errorf("expected 0, got %v err=%v", d, err)
	}
}

func TestParseInterval_Invalid(t *testing.T) {
	_, err := parseInterval("banana")
	if err == nil {
		t.Error("expected error for invalid interval")
	}
}

// ─── isWithinActiveHours ──────────────────────────────────────────────────────

func makeTime(hour, minute int) time.Time {
	return time.Date(2026, 1, 1, hour, minute, 0, 0, time.UTC)
}

func TestActiveHours_InWindow(t *testing.T) {
	h := &HoursRange{Start: "08:00", End: "22:00", Timezone: "UTC"}
	if !isWithinActiveHours(h, makeTime(12, 0)) {
		t.Error("noon should be within 08:00-22:00")
	}
}

func TestActiveHours_OutsideWindow_Before(t *testing.T) {
	h := &HoursRange{Start: "08:00", End: "22:00", Timezone: "UTC"}
	if isWithinActiveHours(h, makeTime(7, 59)) {
		t.Error("07:59 should be outside 08:00-22:00")
	}
}

func TestActiveHours_OutsideWindow_After(t *testing.T) {
	h := &HoursRange{Start: "08:00", End: "22:00", Timezone: "UTC"}
	if isWithinActiveHours(h, makeTime(22, 0)) {
		t.Error("22:00 should be outside (exclusive end) 08:00-22:00")
	}
}

func TestActiveHours_EndAt2400(t *testing.T) {
	h := &HoursRange{Start: "00:00", End: "24:00", Timezone: "UTC"}
	if !isWithinActiveHours(h, makeTime(23, 59)) {
		t.Error("23:59 should be inside 00:00-24:00")
	}
}

// ─── maybeFire ────────────────────────────────────────────────────────────────

func TestMaybeFire_BusyInbox(t *testing.T) {
	q := queue.New(8)
	q.Enqueue("existing", "", nil)

	p := &HeartbeatPlugin{
		cfg:        HeartbeatConfig{Every: "1m", AckMaxChars: 300},
		workingDir: t.TempDir(),
		inbox:      q,
	}
	initialLen := q.Len()
	p.maybeFire(time.Now())
	if q.Len() != initialLen {
		t.Error("maybeFire should skip when inbox is busy")
	}
}

func TestMaybeFire_EffectivelyEmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "HEARTBEAT.md"), []byte("# heading\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	q := queue.New(8)
	p := &HeartbeatPlugin{
		cfg:        HeartbeatConfig{Every: "1m", AckMaxChars: 300},
		workingDir: dir,
		inbox:      q,
	}
	p.maybeFire(time.Now())
	if q.Len() != 0 {
		t.Error("maybeFire should skip when HEARTBEAT.md is effectively empty")
	}
}

func TestMaybeFire_DefaultHeartbeatFile(t *testing.T) {
	dir := t.TempDir()
	q := queue.New(8)
	p := &HeartbeatPlugin{
		cfg:        HeartbeatConfig{Every: "1m", AckMaxChars: 300},
		workingDir: dir,
		inbox:      q,
	}
	p.SetWorkingDir(dir)
	data, err := os.ReadFile(filepath.Join(dir, "HEARTBEAT.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "outstanding tasks") {
		t.Fatalf("unexpected default HEARTBEAT.md: %s", data)
	}
	now := time.Now()
	p.maybeFire(now)
	if q.Len() != 1 {
		t.Errorf("maybeFire should fire when default HEARTBEAT.md exists, got len=%d", q.Len())
	}
	msg := <-q.C()
	if !strings.HasPrefix(msg.ChatId, "heartbeat-") {
		t.Errorf("expected ChatId to have heartbeat- prefix, got %q", msg.ChatId)
	}
}

func TestEnsureDefaultHeartbeatFile_CreatesWhenMissing(t *testing.T) {
	dir := t.TempDir()
	created, err := ensureDefaultHeartbeatFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Error("expected created=true when file is missing")
	}
	b, err := os.ReadFile(filepath.Join(dir, "HEARTBEAT.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "outstanding tasks") {
		t.Fatalf("unexpected body: %s", b)
	}
}

func TestEnsureDefaultHeartbeatFile_NoOverwrite(t *testing.T) {
	dir := t.TempDir()
	custom := "# Custom\n- Keep me\n"
	path := filepath.Join(dir, "HEARTBEAT.md")
	if err := os.WriteFile(path, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	created, err := ensureDefaultHeartbeatFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Error("should not replace existing HEARTBEAT.md")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != custom {
		t.Errorf("expected content preserved, got %q", b)
	}
}
