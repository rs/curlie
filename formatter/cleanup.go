package formatter

import (
	"bytes"
	"io"
)

// HeaderCleaner removes > and < from curl --verbose output.
type HeaderCleaner struct {
	Out io.Writer

	// Verbose removes the request headers part the output as well as the lines
	// starting with * if set to false.
	Verbose bool

	// Post is inserted after the request headers.
	Post *bytes.Buffer

	inited   bool
	muted    bool
	buf      []byte
	last     byte
	skip     int
	skipLine bool
	body     bool
}

func (c *HeaderCleaner) Write(p []byte) (n int, err error) {
	if !c.inited {
		c.inited = true
		c.muted = !c.Verbose
	}
	cp := c.buf
	for i := 0; i < len(p); i++ {
		b := p[i]
		if c.skipLine && b != '\n' {
			continue
		}
		c.skipLine = false
		if c.skip > 0 {
			c.skip--
			continue
		}
		switch b {
		case '>', '<':
			if c.last == '\n' {
				c.skip = 1 // space
				c.last = b
				continue
			}
		case '\r':
			if c.last == '>' {
				c.body = true
				if c.muted {
					c.muted = false
					c.skip = 1
					c.last = '\n'
					continue
				}
			}
		default:
			if c.last == '\n' || c.last == 0 {
				switch b {
				case '{', '}':
					c.skipLine = true
					c.skip = 1
					continue
				case '*':
					if !c.Verbose {
						c.skipLine = true
						c.skip = 1
						continue
					}
					if c.Post != nil && c.body {
						cp = append(append(cp, c.Post.Bytes()...), '\n')
						c.Post = nil
					}
				}
			}
		}
		if !c.muted {
			cp = append(cp, b)
		}
		c.last = b
	}
	if len(cp) > 0 {
		n, err = c.Out.Write(cp)
		if err != nil || n != len(cp) {
			return
		}
	}
	return len(p), nil
}
