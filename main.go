package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/rs/curl-httpie/args"
	"github.com/rs/curl-httpie/formatter"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	var stdout io.Writer = os.Stdout
	var stderr io.Writer = os.Stderr
	var stdin io.Reader = os.Stdin
	input := &bytes.Buffer{}
	var inputWriter io.Writer = input
	opts := args.Parse(os.Args)

	verbose := opts.Has("verbose") || opts.Has("v")
	quiet := opts.Has("silent") || opts.Has("s")
	pretty := opts.Remove("pretty")

	if len(opts) == 0 {
		// Show help if no args
		opts = append(opts, "-h")
	} else {
		// Remove progress bar.
		opts = append(opts, "-sS")
	}

	// Change default method based on binary name.
	switch os.Args[0] {
	case "post", "put", "delete", "head":
		if !opts.Has("X") && !opts.Has("request") {
			opts = append(opts, "-X", os.Args[0])
		}
	}

	if opts.Has("h") || opts.Has("help") {
		stdout = &formatter.HelpAdapter{Out: stdout, CmdName: os.Args[0]}
	} else {
		if pretty || terminal.IsTerminal(1) {
			inputWriter = &formatter.JSON{
				Out:    inputWriter,
				Scheme: formatter.DefaultColorScheme,
			}
			// Format/colorize JSON output if stdout is to the terminal.
			stdout = &formatter.JSON{
				Out:    stdout,
				Scheme: formatter.DefaultColorScheme,
			}
		}
		if pretty || terminal.IsTerminal(2) {
			// If stderr is not redirected, output headers.
			if !quiet {
				opts = append(opts, "-v")
			}
			stderr = &formatter.HeaderColorizer{
				Out:    stderr,
				Scheme: formatter.DefaultColorScheme,
			}
		}
		if data := opts.Val("d"); data != "" {
			// If data is provided via -d, read it from there for the verbose mode.
			// XXX handle the @filename case.
			inputWriter.Write([]byte(data))
		} else if !terminal.IsTerminal(0) {
			// If something is piped in to the command, tell curl to use it as input.
			opts = append(opts, "-d@-")
			// Tee the stdin to the buffer used show the posted data in verbose mode.
			stdin = io.TeeReader(stdin, inputWriter)
		}
	}
	if opts.Has("curl") {
		opts.Remove("curl")
		fmt.Printf("curl %s\n", strings.Join(opts, " "))
	}
	cmd := exec.Command("curl", opts...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = &formatter.HeaderCleaner{
		Out:     stderr,
		Verbose: verbose,
		Post:    input,
	}
	status := 0
	if err := cmd.Run(); err != nil {
		switch err := err.(type) {
		case *exec.ExitError:
			if err.Stderr != nil {
				fmt.Fprint(stderr, string(err.Stderr))
			}
			if ws, ok := err.ProcessState.Sys().(syscall.WaitStatus); ok {
				status = ws.ExitStatus()
			}
		default:
			fmt.Fprint(stderr, err)
		}
	}
	os.Exit(status)
}
