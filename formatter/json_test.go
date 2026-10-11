package formatter

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSONWrite_SanitizesEscapeSequence(t *testing.T) {
	buf := &bytes.Buffer{}
	j := &JSON{
		Out:    buf,
		Scheme: ColorScheme{},
	}
	input := []byte("{\"message\": \"hello \x1b[31mworld\x1b[0m\"}\n")
	_, err := j.Write(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "\x1b") {
		t.Errorf("expected ANSI escape character (0x1b) to be escaped, got raw escape in output: %q", out)
	}
	if !strings.Contains(out, `\u001b[31mworld\u001b[0m`) {
		t.Errorf("expected escaped \\u001b sequence in output, got: %q", out)
	}
}

func TestJSONWrite_NormalFormatting(t *testing.T) {
	buf := &bytes.Buffer{}
	j := &JSON{
		Out:    buf,
		Scheme: ColorScheme{},
	}
	input := []byte(`{"name": "test", "active": true, "count": 42}`)
	_, err := j.Write(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"name": "test"`) {
		t.Errorf("expected formatted JSON with name field, got: %q", out)
	}
}
