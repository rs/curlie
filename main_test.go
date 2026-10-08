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

func TestJSONTransferRequestBodies(t *testing.T) {
	const stdin = `{"piped":true}`
	const first = `{"first":1}`
	const last = `{"last":2}`
	for _, tt := range []struct {
		name     string
		groups   [][]string
		boundary string
		bodies   []string
	}{
		{"JSON then stdin", [][]string{{"--json", first}, {}}, "--next", []string{first, stdin}},
		{"expanded JSON then stdin", [][]string{{"--variable", "body=" + first, "--expand-json", "{{body}}"}, {}}, "--next", []string{first, stdin}},
		{"stdin then JSON", [][]string{{}, {"--json", last}}, "--next", []string{"", last}},
		{"stdin then expanded JSON", [][]string{{}, {"--variable", "body=" + last, "--expand-json", "{{body}}"}}, "--next", []string{"", last}},
		{"two JSON then stdin", [][]string{{"--json", first}, {"--json", last}, {}}, "--next", []string{first, last, stdin}},
		{"middle JSON then stdin", [][]string{{}, {"--json", first}, {}}, "--next", []string{"", first, stdin}},
		{"short next", [][]string{{"--json", first}, {}}, "-:", []string{first, stdin}},
		{"compound short next", [][]string{{"--json", first}, {}}, "-s:", []string{first, stdin}},
		{"next as JSON body", [][]string{{}, {"--json", "--next"}}, "--next", []string{"", "--next"}},
		{"short next as JSON body", [][]string{{}, {"--json", "-:"}}, "--next", []string{"", "-:"}},
		{"next as earlier JSON body", [][]string{{"--json", "--next"}, {}}, "--next", []string{"--next", stdin}},
		{"next as output filename", [][]string{{"--json", first}, {"--json", last, "--output", "--next"}}, "--next", []string{first, last}},
		{"next-like filename before boundary", [][]string{{"--json", first, "--output", "--next"}, {}}, "--next", []string{first, stdin}},
		{"next as header value", [][]string{{"--json", first}, {"--json", last, "--header", "--next"}}, "--next", []string{first, last}},
		{"short next as user agent", [][]string{{"--json", first}, {"--json", last, "-A", "-:"}}, "--next", []string{first, last}},
		{"compound next-like user agent", [][]string{{"--json", first}, {"--json", last, "-A--next"}}, "--next", []string{first, last}},
		{"JSON-like filename after boundary", [][]string{{"--json", first}, {"--output", "--json"}}, "--next", []string{first, stdin}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			type request struct{ path, method, body string }
			requests := make(chan request, len(tt.groups))
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				requests <- request{r.URL.Path, r.Method, string(body)}
				fmt.Fprint(w, "{}")
			}))
			defer server.Close()
			argv := []string{"--disable"}
			for i, group := range tt.groups {
				if i > 0 {
					argv = append(argv, tt.boundary)
				}
				argv = append(argv, "--noproxy", "*", "--max-time", "5")
				argv = append(argv, group...)
				// Explicit --url preserves curl's transfer groups through Parse.
				argv = append(argv, "--url", fmt.Sprintf("%s/%d", server.URL, i))
			}
			runCurlie(t, argv, stdin, t.TempDir())
			for i, body := range tt.bodies {
				method := "POST"
				if body == "" {
					method = "GET"
				}
				want := request{fmt.Sprintf("/%d", i), method, body}
				select {
				case got := <-requests:
					if got != want {
						t.Errorf("request %d = %#v, want %#v", i, got, want)
					}
				default:
					t.Fatalf("no request received for transfer %d", i)
				}
			}
		})
	}
}

func TestJSONTransferEqualsArguments(t *testing.T) {
	// Check forwarding for equals syntax even when native curl is too old to accept it.
	for _, tt := range []struct {
		name     string
		options  []string
		addStdin bool
	}{
		{"first JSON", []string{"--json={}", "--url", "http://127.0.0.1:1/first", "--next", "--url", "http://127.0.0.1:1/last"}, true},
		{"first expanded JSON", []string{"--expand-json={}", "--url", "http://127.0.0.1:1/first", "-:", "--url", "http://127.0.0.1:1/last"}, true},
		{"last JSON", []string{"--url", "http://127.0.0.1:1/first", "--next", "--json={}", "--url", "http://127.0.0.1:1/last"}, false},
		{"last expanded JSON", []string{"--url", "http://127.0.0.1:1/first", "--next", "--expand-json={}", "--url", "http://127.0.0.1:1/last"}, false},
		{"equals JSON next-like body", []string{"--json=--next", "--url", "http://127.0.0.1:1/"}, false},
		{"equals expanded JSON next-like body", []string{"--expand-json=-:", "--url", "http://127.0.0.1:1/"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			argv := append([]string{"--disable", "--curl"}, tt.options...)
			output := runCurlie(t, argv, "{}", t.TempDir())
			if strings.Contains(output, " -d@-") != tt.addStdin {
				t.Fatalf("unexpected curl arguments: %s", output)
			}
		})
	}
}
