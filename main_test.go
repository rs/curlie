package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCurlieProcess(t *testing.T) {
	if os.Getenv("CURLIE_TEST_PROCESS") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"curlie"}, os.Args[i+1:]...)
			main()
			return
		}
	}
	t.Fatal("missing process arguments")
}

func runCurlie(t *testing.T, argv []string, input, cwd string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestCurlieProcess$", "--"}, argv...)...)
	cmd.Env = append(os.Environ(), "CURLIE_TEST_PROCESS=1")
	cmd.Dir = cwd
	cmd.Stdin = strings.NewReader(input)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("curlie failed: %v\n%s", err, output)
	}
	return string(output)
}

func TestJSONRequestBody(t *testing.T) {
	const payload = `{"input":"test"}`
	for _, tt := range []struct {
		name, stdin, body string
		options           []string
	}{
		{"stdin", payload, payload, []string{"--json", "@-"}},
		{"inline", "ignored", payload, []string{"--json", payload}},
		{"file", "ignored", payload, []string{"--json", "@payload.json"}},
		{"repeated", "ignored", "{}", []string{"--json", "{", "--json", "}"}},
		{"empty JSON", "ignored", "", []string{"--json", ""}},
		{"empty stdin", "", "", []string{"--json", "@-"}},
		{"expanded JSON", "ignored", payload, []string{"--variable", "body=" + payload, "--expand-json", "{{body}}"}},
		{"ordinary stdin", payload, payload, nil},
		{"JSON-like output filename", payload, payload, []string{"--output", "--json"}},
		{"JSON-like equals output filename", payload, payload, []string{"--output", "--json=output"}},
		{"JSON-like header value", payload, payload, []string{"--header", "--json"}},
		{"expanded JSON-like output filename", payload, payload, []string{"--output", "--expand-json"}},
		{"expanded JSON-like equals output filename", payload, payload, []string{"--output", "--expand-json=output"}},
		{"expanded JSON-like header value", payload, payload, []string{"--header", "--expand-json"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cwd := t.TempDir()
			if err := os.WriteFile(filepath.Join(cwd, "payload.json"), []byte(payload), 0600); err != nil {
				t.Fatal(err)
			}
			bodies := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				bodies <- string(body)
				fmt.Fprint(w, "{}")
			}))
			defer server.Close()
			// Disable curlrc before any other curl option, and keep all traffic on loopback.
			argv := append([]string{"--disable", "--noproxy", "*", "--max-time", "5"}, tt.options...)
			argv = append(argv, server.URL)
			runCurlie(t, argv, tt.stdin, cwd)
			select {
			case body := <-bodies:
				if body != tt.body {
					t.Fatalf("request body = %q, want %q", body, tt.body)
				}
			default:
				t.Fatal("no request received")
			}
		})
	}
}

func TestJSONEqualsDoesNotAddStdinArgument(t *testing.T) {
	// curl versions before 8.16 reject equals syntax, so check the forwarded argv
	// without sending a request instead of requiring a newer native curl.
	for _, option := range []string{"--json=@-", "--expand-json={{body}}"} {
		t.Run(option, func(t *testing.T) {
			output := runCurlie(t, []string{"--disable", "--curl", option, "http://127.0.0.1:1/"}, "{}", t.TempDir())
			if !strings.Contains(output, " "+option+" ") || strings.Contains(output, " -d@-") {
				t.Fatalf("unexpected curl arguments: %s", output)
			}
		})
	}
}
