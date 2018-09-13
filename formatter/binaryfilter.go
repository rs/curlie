package formatter

import (
	"bytes"
	"io"
	"strings"
)

type BinaryFilter struct {
	Out    io.Writer
	ignore bool
}

var binarySuppressNotice = []byte(strings.Join([]string{
	"+-----------------------------------------+",
	"| NOTE: binary data not shown in terminal |",
	"+-----------------------------------------+\n",
}, "\n"))

func (bf *BinaryFilter) Write(p []byte) (n int, err error) {
	if bf.ignore {
		return len(p), nil
	}
	if bytes.IndexByte(p, 0) != -1 {
		bf.ignore = true
		bf.Out.Write(binarySuppressNotice)
		return len(p), nil
	}
	return bf.Out.Write(p)
}
