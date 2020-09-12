// +build windows

// Package terminal provides an alternate version Go's IsTerminal,
// as a workaround for https://github.com/rs/curlie/issues/12
//
// On Windows the very act of checking Stdin causes Stdin to open.
// Since we may need Stdin, it is best to not check it and just
// assume that the user is expecting pretty output, or will disable
// it, or will use curl.exe.
//
// TODO However, there may be a better solution:
//   1. Check if Stdin is open (or will be opened by curl/curlie)
//   2. Check IsTerminal(0) (already knowing the open/close state)
//   3. Stdin.Close(), unless it was open, or will be opened
// See https://stackoverflow.com/a/38612652/151312
package terminal

import (
	"golang.org/x/sys/windows"
)

var fdStdin = 0

// IsTerminal returns whether the given file descriptor is a terminal.
func IsTerminal(fd int) bool {
	if fdStdin == fd {
		// This is a hack. See the note above to help fix this.
		return true
	}

	var st uint32
	err := windows.GetConsoleMode(windows.Handle(fd), &st)
	return err == nil
}