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

var (
	capath  = []byte("  CApath:")
	ccapath = []byte("*   CApath:")
)

func (c *HeaderCleaner) Write(p []byte) (n int, err error) {
	n = len(p)
	cp := c.buf
	p = bytes.Replace(p, capath, ccapath, 1) // Fix curl misformatted line
	for len(p) > 0 {
		idx := bytes.IndexByte(p, '\n')
		if idx == -1 {
			c.line = append(c.line, p...)
			break
		}
		c.line = append(c.line, p[:idx+1]...)
		p = p[idx+1:]
		cp = c.processLine(c.line, cp)
		c.line = c.line[:0]
	}
	_, err = c.Out.Write(cp)
	return
}

var colorEscape = []byte("\x1b[")

func firstVisibleChar(b []byte) (byte, int) {
	if bytes.HasPrefix(b, colorEscape) {
		if idx := bytes.IndexByte(b, 'm'); idx != -1 {
			return b[idx], idx + 1
			/*
				if idx < len(b) {
					return b[idx+1], idx + 1
				} else {
					return 0, -1
				}
			*/
		}
	}
	return b[0], 0
}

func (c *HeaderCleaner) processLine(
	line []byte,
	output []byte,
) []byte {
	ignore := false
	b, i := firstVisibleChar(line)

	switch b {
	case '>':
		if c.Verbose {
			line = line[i+2:]
		} else {
			ignore = true
		}
	case '<':
		prefixEnd := i + 2
		if prefixEnd <= len(line)-1 {
			line = line[prefixEnd:]
		} else {
			line = line[prefixEnd-1:]
		}
	case '}', '{':
		ignore = true
		if c.Verbose && c.Post != nil {
			output = append(
				output,
				bytes.TrimSpace(c.Post.Bytes())...,
			)
			output = append(output, '\n', '\n')
			c.Post = nil
		}
	case '*':
		if !c.Verbose {
			ignore = true
		}
	}

	if !ignore {
		output = append(output, line...)
	}
	return output
}
