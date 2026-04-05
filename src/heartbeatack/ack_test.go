package heartbeatack

import "testing"

func TestStripAck_TokenAtStart(t *testing.T) {
	_, suppress := stripAck("HEARTBEAT_OK", 300)
	if !suppress {
		t.Error("bare HEARTBEAT_OK should be suppressed")
	}
}

func TestStripAck_TokenAtEnd(t *testing.T) {
	_, suppress := stripAck("All good. HEARTBEAT_OK", 300)
	if !suppress {
		t.Error("token at end with short prefix should be suppressed")
	}
}

func TestStripAck_TokenInMiddle(t *testing.T) {
	long := "This message says HEARTBEAT_OK and then continues with more content that exceeds the limit considerably."
	_, suppress := stripAck(long, 10)
	if suppress {
		t.Error("token in the middle with long surrounding text should not be suppressed")
	}
}

func TestStripAck_MaxCharsExceeded(t *testing.T) {
	long := "HEARTBEAT_OK " + string(make([]byte, 400))
	_, suppress := stripAck(long, 300)
	if suppress {
		t.Error("remainder exceeding maxAckChars should not be suppressed")
	}
}

func TestStripAck_NoToken(t *testing.T) {
	_, suppress := stripAck("Just a regular reply.", 300)
	if suppress {
		t.Error("reply without HEARTBEAT_OK should not be suppressed")
	}
}

func TestShouldSuppressAckReply_MatchesStripAck(t *testing.T) {
	cases := []struct {
		raw    string
		max    int
		expect bool
	}{
		{"HEARTBEAT_OK", 300, true},
		{"Just a regular reply.", 300, false},
	}
	for _, tc := range cases {
		_, fromStrip := stripAck(tc.raw, tc.max)
		fromExport := ShouldSuppressAckReply(tc.raw, tc.max)
		if fromStrip != fromExport {
			t.Errorf("%q: stripAck=%v ShouldSuppressAckReply=%v", tc.raw, fromStrip, fromExport)
		}
	}
}
