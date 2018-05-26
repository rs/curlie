package formatter

import (
	"io"
	"regexp"
)

// ColorScheme contains coloring configuration for the formatters.
type ColorScheme struct {
	Default string
	Comment string
	Status  string
	Field   string
	Value   string
	Literal string
}

type ColorName int

const (
	DefaultColor ColorName = iota
	CommentColor
	StatusColor
	FieldColor
	ValueColor
	LiteralColor
)

func (cs ColorScheme) Color(name ColorName) string {
	switch name {
	case DefaultColor:
		return cs.Default
	case CommentColor:
		return cs.Comment
	case StatusColor:
		return cs.Status
	case FieldColor:
		return cs.Field
	case ValueColor:
		return cs.Value
	case LiteralColor:
		return cs.Literal
	}
	return ""
}

func (cs ColorScheme) IsZero() bool {
	return cs == ColorScheme{}
}

var DefaultColorScheme = ColorScheme{
	Default: "\x1b[38;5;245m",
	Comment: "\x1b[38;5;237m",
	Status:  "\x1b[38;5;136m",
	Field:   "\x1b[38;5;33m",
	Value:   "\x1b[38;5;37m",
	Literal: "\x1b[38;5;166m",
}

var reset = "\x1b[39m"

type HeaderColorizer struct {
	Out    io.Writer
	Scheme ColorScheme
	buf    []byte
	line   []byte
}

func (c *HeaderColorizer) Write(p []byte) (n int, err error) {
	c.buf = c.buf[:0]
	for i := 0; i < len(p); i++ {
		b := p[i]
		c.line = append(c.line, b)
		if b == '\n' {
			c.formatLine()
			continue
		}
	}
	n, err = c.Out.Write(c.buf)
	if err != nil || n != len(c.buf) {
		return
	}
	return len(p), nil
}

type headerFormatter struct {
	re     *regexp.Regexp
	colors []ColorName
}

var headerFormatters = []headerFormatter{
	{
		regexp.MustCompile(`^([A-Z]+)(\s+\S+\s+)(HTTP)(/)([\d\.]+\s*\n)$`),
		[]ColorName{FieldColor, DefaultColor, FieldColor, DefaultColor, ValueColor},
	},
	{
		regexp.MustCompile(`^(HTTP)(/)([\d.]+\s+\d{3})(\s+.+\n)$`),
		[]ColorName{FieldColor, DefaultColor, ValueColor, StatusColor},
	},
	{
		regexp.MustCompile(`^([a-zA-Z0-9.-]*?:)(.*\n)$`),
		[]ColorName{DefaultColor, ValueColor},
	},
	{
		regexp.MustCompile(`^(\* .*[\n\r]*)$`),
		[]ColorName{CommentColor},
	},
}

func (c *HeaderColorizer) formatLine() {
	defer func() {
		c.line = c.line[:0]
		c.buf = append(c.buf, reset...)
	}()
	cs := c.Scheme
	if cs.IsZero() {
		c.buf = append(c.buf, c.line...)
		return
	}
	for _, formatter := range headerFormatters {
		m := formatter.re.FindSubmatch(c.line)
		if m == nil {
			continue
		}
		for i, s := range m[1:] {
			col := cs.Color(formatter.colors[i])
			c.buf = append(append(c.buf, col...), s...)
		}
		return
	}
	c.buf = append(c.buf, c.line...)
}
