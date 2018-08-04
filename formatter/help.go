package formatter

import (
	"bytes"
	"io"
)

type HelpAdapter struct {
	Out     io.Writer
	CmdName string
}

func (j HelpAdapter) Write(p []byte) (n int, err error) {
	n = len(p)
	cmd := "http"
	if len(j.CmdName) == 0 {
		cmd = j.CmdName
	}
	p = bytes.Replace(p,
		[]byte("curl [options...] <url>"),
		[]byte(cmd+" [options...] [METHOD] URL [REQUEST_ITEM [REQUEST_ITEM ...]]"), 1)
	_, err = j.Out.Write(p)
	return
}
