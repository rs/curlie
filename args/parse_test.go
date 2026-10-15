package args

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	expected := []string{"example.com"}
	parseAndCompare(t, []string{"curlie", "example.com"}, expected)
}

func TestParsePost(t *testing.T) {
	expected := []string{"-X", "POST", "example.com"}
	parseAndCompare(t, []string{"curlie", "post", "example.com"}, expected)
}

func TestParseHead(t *testing.T) {
	expected := []string{"-I", "example.com"}
	parseAndCompare(t, []string{"curlie", "head", "example.com"}, expected)
}

func TestParseForm(t *testing.T) {
	expected := []string{"-X", "POST", "--form-string", "name=value", "--form-string", "user=@alice", "--form-string", "note=<item>", "example.com"}
	parseAndCompare(t, []string{"curlie", "--form", "post", "example.com", "name=value", "user=@alice", "note=<item>"}, expected)
}

func TestParseFormFile(t *testing.T) {
	expected := []string{"-X", "POST", "-F", "avatar=@/tmp/x", "example.com"}
	parseAndCompare(t, []string{"curlie", "--form", "post", "example.com", "avatar@/tmp/x"}, expected)
}

func TestParseFormFileType(t *testing.T) {
	expected := []string{"-X", "POST", "-F", "avatar=@/tmp/x.png;type=image/png", "example.com"}
	parseAndCompare(t, []string{"curlie", "--form", "post", "example.com", "avatar@/tmp/x.png;type=image/png"}, expected)
}

func TestParseFormFileAndFields(t *testing.T) {
	expected := []string{"-X", "POST", "--form-string", "name=@notafile", "-F", "avatar=@/tmp/x", "--form-string", "email=me@example.com", "example.com"}
	parseAndCompare(t, []string{"curlie", "--form", "post", "example.com", "name=@notafile", "avatar@/tmp/x", "email=me@example.com"}, expected)
}

func TestParseAtInHeaderAndParam(t *testing.T) {
	expected := []string{"-H", "From:me@example.com", "example.com?q=a%40b"}
	parseAndCompare(t, []string{"curlie", "example.com", "From:me@example.com", "q==a@b"}, expected)
}

func TestParseJSONLiteralAt(t *testing.T) {
	expected := []string{"-X", "POST", "-d", `{"name":"@notafile"}`, "example.com"}
	parseAndCompare(t, []string{"curlie", "post", "example.com", "name=@notafile"}, expected)
}

func TestParseJSONFileRejected(t *testing.T) {
	args := []string{"curlie", "post", "example.com", "name=value", "avatar@/tmp/x"}
	opts, err := Parse(args)
	if err == nil {
		t.Fatalf("Expecting an error for a file field without --form, but got %v for %v", opts, args)
	}
	if !strings.Contains(err.Error(), "--form") || !strings.Contains(err.Error(), "avatar") {
		t.Errorf("Expecting the error to name the field and suggest --form, got %q", err)
	}
}

func parseAndCompare(t *testing.T, args, expected []string) {
	opts, err := Parse(args)
	if err != nil {
		t.Fatalf("Unexpected error %v for %v", err, args)
	}
	if !compareStrings(opts, expected) {
		t.Errorf("Expecting %v, but got %v for %v", expected, opts, args)
	}

}

func compareStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if b[i] != v {
			return false
		}
	}
	return true
}
