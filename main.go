package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/rs/curl-httpie/args"
	"github.com/rs/curl-httpie/formatter"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	var stdout io.Writer = os.Stdout
	var stderr io.Writer = os.Stderr
	var stdin io.Reader = os.Stdin
	input := &bytes.Buffer{}
	opts := args.Parse(os.Args)

	verbose := opts.Has("verbose") || opts.Has("v")
	quiet := opts.Has("silent") || opts.Has("s")

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
		if data := opts.Val("d"); data != "" {
			// If data is provided via -d, read it from there for the verbose mode.
			// XXX handle the @filename case.
			w := &formatter.JSON{
				Out:    input,
				Scheme: formatter.DefaultColorScheme,
			}
			w.Write([]byte(data))
		} else if !terminal.IsTerminal(0) {
			// If something is piped in to the command, tell curl to use it as input.
			opts = append(opts, "-d@-")
			// Tee the stdin to the buffer used show the posted data in verbose mode.
			stdin = io.TeeReader(stdin, &formatter.JSON{
				Out:    input,
				Scheme: formatter.DefaultColorScheme,
			})
		}
		if terminal.IsTerminal(1) {
			// Format/colorize JSON output if stdout is to the terminal.
			stdout = &formatter.JSON{
				Out:    stdout,
				Scheme: formatter.DefaultColorScheme,
			}
		}
		if terminal.IsTerminal(2) {
			// If stderr is not redirected, output headers.
			if !quiet {
				opts = append(opts, "-v")
			}
			stderr = &formatter.HeaderColorizer{
				Out:    stderr,
				Scheme: formatter.DefaultColorScheme,
			}
		}
	}
	cmd := exec.Command("curl", opts...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = &formatter.HeaderCleaner{
		Out:     stderr,
		Verbose: verbose,
		Post:    input,
	}
	if err := cmd.Run(); err != nil {
		fmt.Println(err)
	}
}
