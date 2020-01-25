package args

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type argType int

const (
	unknownArg argType = iota
	headerArg
	paramArg
	fieldArg
	jsonArg
)

func parseFancyArgs(args []string, isForm bool) (opts Opts) {
	if len(args) == 0 {
		return
	}
	method := strings.ToUpper(args[0])
	switch method {
	case "GET", "POST", "PUT", "DELETE":
		opts = append(opts, "-X", method)
		args = args[1:]
	case "HEAD":
		opts = append(opts, "-I")
		args = args[1:]
	}
	if len(args) == 0 {
		return
	}
	url := args[0]
	data := map[string]interface{}{}
	for _, arg := range args[1:] {
		typ, name, value := parseArg(arg)
		switch typ {
		case headerArg:
			opts = append(opts, "-H", name+":"+value)
		case paramArg:
			url = appendURLParam(url, name, value)
		case fieldArg:
			if isForm {
				opts = append(opts, "-F", name+"="+value)
			} else {
				data[name] = value
			}
		case jsonArg:
			var v interface{}
			json.Unmarshal([]byte(value), &v)
			data[name] = v
		default:
			opts = append(opts, arg)
		}
	}
	if len(data) > 0 {
		j, _ := json.Marshal(data)
		opts = append(opts, "-d", string(j))
	}
	opts = append(opts, normalizeURL(url))
	return
}

func normalizeURL(u string) string {
	// If scheme is omitted, use http:
	if !strings.HasPrefix(u, "http") {
		if strings.HasPrefix(u, "//") {
			u = "http:" + u
		} else {
			u = "http://" + u
		}
	}
	pu, err := url.Parse(u)
	if err != nil {
		fmt.Print(err)
		return u
	}
	if pu.Host == ":" {
		pu.Host = "localhost"
	} else if pu.Host != "" && pu.Host[0] == ':' {
		// If :port is given with no hostname, add localhost
		pu.Host = "localhost" + pu.Host
	}
	return pu.String()
}

func parseArg(arg string) (typ argType, name, value string) {
	for i := 0; i < len(arg); i++ {
		switch arg[i] {
		case ':':
			if i+1 < len(arg) && arg[i+1] == '=' {
				return jsonArg, arg[:i], arg[i+2:]
			}
			return headerArg, arg[:i], arg[i+1:]
		case '=':
			if i+1 < len(arg) && arg[i+1] == '=' {
				return paramArg, arg[:i], arg[i+2:]
			}
			return fieldArg, arg[:i], arg[i+1:]
		}
	}
	return
}

func appendURLParam(u, name, value string) string {
	sep := "?"
	if strings.IndexByte(u, '?') != -1 {
		sep = "&"
	}
	return u + sep + url.QueryEscape(name) + "=" + url.QueryEscape(value)
}
