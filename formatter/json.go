package formatter

import (
	"bytes"
	"io"
)

// JSON is a writer that formats/colorizes JSON without decoding it.
// If the stream of bytes does not start with {, the formatting is disabled.
type JSON struct {
	Out       io.Writer
	Scheme    ColorScheme
	inited    bool
	disabled  bool
	last      byte
	lastQuote byte
	isValue   bool
	level     int
	buf       []byte
}

var indent = []byte(`    `)

func (j *JSON) Write(p []byte) (n int, err error) {
	if !j.inited && len(p) > 0 {
		// Only JSON object are supported.
		j.disabled = (p[0] != '{' && p[0] != '[')
		j.inited = true
	}
	if j.disabled {
		return j.Out.Write(p)
	}
	cs := j.Scheme
	cp := j.buf
	for i := 0; i < len(p); i++ {
		b := p[i]
		if j.last == '\\' {
			cp = append(cp, b)
			j.last = b
			continue
		}
		switch b {
		case '\'', '"':
			switch j.lastQuote {
			case 0:
				j.lastQuote = b
				c := cs.Field
				if j.isValue {
					c = cs.Value
				}
				cp = append(append(cp, c...), b)
			case b:
				j.lastQuote = 0
				cp = append(cp, b)
			}
			continue
		default:
			if j.lastQuote != 0 {
				cp = append(cp, b)
				j.last = b
				continue
			}
		}
		switch b {
		case ' ', '\t', '\r', '\n':
			// Skip spaces outside of quoted areas.
			continue
		case '{', '[':
			j.isValue = false
			j.level++
			cp = append(append(append(cp, cs.Default...), b, '\n'), bytes.Repeat(indent, j.level)...)
		case '}', ']':
			j.level--
			if j.level < 0 {
				j.level = 0
			}
			cp = append(append(append(append(cp, '\n'), bytes.Repeat(indent, j.level)...), cs.Default...), b)
			if (p[0] != '}' && p[0] != ']') && j.level == 0 {
				// Add a return after the outer closing brace.
				// If cs is zero that means color is disabled, so only append '\n'
				// else append '\n' and ResetColor.
				if cs.IsZero() {
					cp = append(cp, '\n')
				} else {
					cp = append(append(cp, '\n'), cs.Color(ResetColor)...)
				}

			}
		case ':':
			j.isValue = true
			cp = append(append(cp, cs.Default...), b, ' ')
		case ',':
			j.isValue = false
			cp = append(append(append(cp, cs.Default...), b, '\n'), bytes.Repeat(indent, j.level)...)
		default:
			if j.last == ':' {
				switch b {
				case 'n', 't', 'f':
					// null, true, false
					cp = append(cp, cs.Literal...)
				default:
					// unquoted values like numbers
					cp = append(cp, cs.Value...)
				}
			}
			cp = append(cp, b)
		}
		j.last = b
	}
	n, err = j.Out.Write(cp)
	if err != nil || n != len(cp) {
		return
	}
	return len(p), nil
}
