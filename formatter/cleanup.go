package formatter

import (
	"bytes"
	"io"
)

// HeaderCleaner removes > and < from curl --verbose output.
type HeaderCleaner struct {
	Out io.Writer

	// Verbose removes the request headers part of the output as well as the lines
	// starting with * if set to false.
	Verbose bool

	// Post is inserted after the request headers.
	Post *bytes.Buffer

	buf  []byte
	line []byte
}

func (c *HeaderCleaner) Write(p []byte) (n int, err error) {
	n = len(p)
	cp := c.buf
	for len(p) > 0 {
		idx := bytes.IndexByte(p, '\n')
		if idx == -1 {
			c.line = append(c.line, p...)
			break
		}
		c.line = append(c.line, p[:idx+1]...)
		p = p[idx+1:]
		ignore := false
		b, i := firstVisibleChar(c.line)
		switch b {
		case '>':
			if c.Verbose {
				c.line = c.line[i+2:]
			} else {
				ignore = true
			}
		case '<':
			c.line = c.line[i+2:]
		case '}', '{':
			ignore = true
			if c.Verbose && c.Post != nil {
				cp = append(append(cp, bytes.TrimSpace(c.Post.Bytes())...), '\n', '\n')
				c.Post = nil
			}
		case '*':
			if !c.Verbose {
				ignore = true
			}
		}
		if !ignore {
			cp = append(cp, c.line...)
		}
		c.line = c.line[:0]
	}
	_, err = c.Out.Write(cp)
	return
}

var colorEscape = []byte("\x1b[")

func firstVisibleChar(b []byte) (byte, int) {
	if bytes.HasPrefix(b, colorEscape) {
		if idx := bytes.IndexByte(b, 'm'); idx != -1 {
			if idx < len(b) {
				return b[idx+1], idx + 1
			} else {
				return 0, -1
			}
		}
	}
	return b[0], 0
}
