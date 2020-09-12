// +build !windows

package terminal

import (
	"golang.org/x/crypto/ssh/terminal"
)

// IsTerminal returns whether the given file descriptor is a terminal.
func IsTerminal(fd int) bool {
	return terminal.IsTerminal(fd)
}
