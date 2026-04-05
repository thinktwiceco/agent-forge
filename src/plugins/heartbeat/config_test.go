package heartbeat

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	d := DefaultConfig()
	if d.Every != "30m" || d.AckMaxChars != 300 {
		t.Fatalf("DefaultConfig() = %#v", d)
	}
	if d.Prompt != "" || d.ActiveHours != nil {
		t.Fatalf("expected empty prompt and nil active hours, got %#v", d)
	}
}

func TestMergeConfig_Nil(t *testing.T) {
	got := MergeConfig(nil)
	want := DefaultConfig()
	if got != want {
		t.Fatalf("MergeConfig(nil) = %#v, want %#v", got, want)
	}
}

func TestMergeConfig_EmptyStruct(t *testing.T) {
	got := MergeConfig(&HeartbeatConfig{})
	want := DefaultConfig()
	if got != want {
		t.Fatalf("MergeConfig(&{}) = %#v, want %#v", got, want)
	}
}

func TestMergeConfig_PartialEvery(t *testing.T) {
	got := MergeConfig(&HeartbeatConfig{Every: "15m"})
	if got.Every != "15m" || got.AckMaxChars != 300 {
		t.Fatalf("got %#v", got)
	}
}

func TestMergeConfig_ExplicitDisable(t *testing.T) {
	got := MergeConfig(&HeartbeatConfig{Every: "0m"})
	if got.Every != "0m" {
		t.Fatalf("expected Every 0m, got %#v", got)
	}
}

func TestMergeConfig_PromptAndAck(t *testing.T) {
	got := MergeConfig(&HeartbeatConfig{
		Prompt:      "Custom prompt",
		AckMaxChars: 100,
	})
	if got.Prompt != "Custom prompt" || got.AckMaxChars != 100 {
		t.Fatalf("got %#v", got)
	}
	if got.Every != "30m" {
		t.Fatalf("Every should stay default, got %q", got.Every)
	}
}

func TestMergeConfig_ActiveHours(t *testing.T) {
	h := &HoursRange{Start: "09:00", End: "17:00", Timezone: "UTC"}
	got := MergeConfig(&HeartbeatConfig{ActiveHours: h})
	if got.ActiveHours != h {
		t.Fatalf("got %#v", got.ActiveHours)
	}
}
