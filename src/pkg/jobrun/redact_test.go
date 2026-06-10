package jobrun

import (
	"strings"
	"testing"
)

func TestRedactText(t *testing.T) {
	raw := `Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.token {"password":"admin888","api_key":"sk-live-secret","query":"ok"}`
	out := RedactText(raw)
	if strings.Contains(out, "admin888") || strings.Contains(out, "sk-live-secret") || strings.Contains(out, "eyJhbGci") {
		t.Fatalf("expected secrets redacted, got %q", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("expected redaction marker, got %q", out)
	}
}

func TestSummarizePayloadRedactsJSON(t *testing.T) {
	out := SummarizePayload(`{"token":"abc123","message":"hello"}`)
	if strings.Contains(out, "abc123") {
		t.Fatalf("expected token redacted, got %q", out)
	}
}
