package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/rs/curlie/args"
	"github.com/rs/curlie/formatter"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	commit  = "0000000"
	version = "v0.0.0-LOCAL"
	date    = "0000-00-00T00:00:00Z"
)

func main() {
	// handle `curlie version` separately from `curl --version`
	if 2 == len(os.Args) && "version" == os.Args[1] {
		fmt.Printf("curlie %s (%s)\n", version, date)
		os.Exit(0)
		return
	}

	// *nixes use 0, 1, 2
	// Windows uses random numbers
	stdinFd := int(os.Stdin.Fd())
	stdoutFd := int(os.Stdout.Fd())
	stderrFd := int(os.Stderr.Fd())

	var stdout io.Writer = os.Stdout
	var stderr io.Writer = os.Stderr
	var stdin io.Reader = os.Stdin
	input := &bytes.Buffer{}
	var inputWriter io.Writer = input
	opts := args.Parse(os.Args)

	verbose := opts.Has("verbose") || opts.Has("v")
	quiet := opts.Has("silent") || opts.Has("s")
	pretty := opts.Remove("pretty")
	opts.Remove("i")

	if len(opts) == 0 {
		// Show help if no args
		opts = append(opts, "-h", "all")
	} else {
		// Remove progress bar.
		opts = append(opts, "-s", "-S")
	}

	// Change default method based on binary name.
	switch os.Args[0] {
	case "post", "put", "delete":
		if !opts.Has("X") && !opts.Has("request") {
			opts = append(opts, "-X", os.Args[0])
		}
	case "head":
		if !opts.Has("I") && !opts.Has("head") {
			opts = append(opts, "-I")
		}
	}

	if opts.Has("h") || opts.Has("help") {
		stdout = &formatter.HelpAdapter{Out: stdout, CmdName: os.Args[0]}
	} else {
		isForm := opts.Has("F")
		if pretty || terminal.IsTerminal(stdoutFd) {
			inputWriter = &formatter.JSON{
				Out:    inputWriter,
				Scheme: formatter.DefaultColorScheme,
			}
			// Format/colorize JSON output if stdout is to the terminal.
			stdout = &formatter.JSON{
				Out:    stdout,
				Scheme: formatter.DefaultColorScheme,
			}
			// Filter out binary output.
			stdout = &formatter.BinaryFilter{Out: stdout}
		}
		if pretty || terminal.IsTerminal(stderrFd) {
			// If stderr is not redirected, output headers.
			if !quiet {
				opts = append(opts, "-v")
			}
			stderr = &formatter.HeaderColorizer{
				Out:    stderr,
				Scheme: formatter.DefaultColorScheme,
			}
		}
		hasInput := true
		if data := opts.Val("d"); data != "" {
			// If data is provided via -d, read it from there for the verbose mode.
			// XXX handle the @filename case.
			inputWriter.Write([]byte(data))
		} else if !terminal.IsTerminal(stdinFd) {
			// If something is piped in to the command, tell curl to use it as input.
			opts = append(opts, "-d@-")
			// Tee the stdin to the buffer used show the posted data in verbose mode.
			stdin = io.TeeReader(stdin, inputWriter)
		} else {
			hasInput = false
		}
		if hasInput {
			if !headerSupplied(opts, "Content-Type") && !isForm {
				opts = append(opts, "-H", "Content-Type: application/json")
			}
		}
	}
	if !headerSupplied(opts, "Accept") {
		opts = append(opts, "-H", "Accept: application/json, */*")
	}
	if opts.Has("curl") {
		opts.Remove("curl")
		fmt.Print("curl")
		for _, opt := range opts {
			if strings.IndexByte(opt, ' ') != -1 {
				fmt.Printf(" %q", opt)
			} else {
				fmt.Printf(" %s", opt)
			}
		}
		fmt.Println()
		return
	}
	cmd := exec.Command("curl", opts...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = &formatter.HeaderCleaner{
		Out:     stderr,
		Verbose: verbose,
		Post:    input,
	}
	if (opts.Has("I") || opts.Has("head")) && terminal.IsTerminal(stdoutFd) {
		cmd.Stdout = ioutil.Discard
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

func headerSupplied(opts args.Opts, header string) bool {
	header = strings.ToLower(header)
	for _, h := range append(opts.Vals("H"), opts.Vals("header")...) {
		if strings.HasPrefix(strings.ToLower(h), header+":") {
			return true
		}
	}
	return false
}
