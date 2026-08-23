package webapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateLogTail(t *testing.T) {
	for _, test := range []struct {
		query string
		want  string
		valid bool
	}{
		{"", "200", true},
		{"?tail=1", "1", true},
		{"?tail=5000", "5000", true},
		{"?tail=0", "", false},
		{"?tail=5001", "", false},
		{"?tail=all", "", false},
	} {
		r := httptest.NewRequest("GET", "/logs"+test.query, nil)
		got, err := validateLogTail(r)
		if test.valid && err != nil {
			t.Fatalf("validateLogTail(%q): %v", test.query, err)
		}
		if !test.valid && err == nil {
			t.Fatalf("validateLogTail(%q) unexpectedly succeeded", test.query)
		}
		if got != test.want {
			t.Fatalf("validateLogTail(%q) = %q, want %q", test.query, got, test.want)
		}
	}
}

func TestSendLogEventEscapesMultilinePayload(t *testing.T) {
	recorder := httptest.NewRecorder()
	if err := sendLogEvent(recorder, recorder, "log", logEvent{Chunk: "first\nsecond"}); err != nil {
		t.Fatal(err)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `data: {"chunk":"first\nsecond"}`) {
		t.Fatalf("unexpected SSE body: %q", body)
	}
}
