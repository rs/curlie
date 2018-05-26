package args

import (
	"encoding/json"
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

func parseFancyArgs(args []string) (opts []string) {
	if len(args) == 0 {
		return
	}
	method := strings.ToUpper(args[0])
	switch method {
	case "HEAD", "GET", "POST", "PUT", "DELETE":
		opts = append(opts, "-X", method)
		args = args[1:]
	}
	if len(args) == 0 {
		return
	}
	hostIdx := len(opts)
	opts = append(opts, args[0]) // host
	args = args[1:]
	data := map[string]interface{}{}
	for _, arg := range args {
		typ, name, value := parseArg(arg)
		switch typ {
		case headerArg:
			opts = append(opts, "-H", name+":"+value)
		case paramArg:
			opts[hostIdx] = appendURLParam(opts[hostIdx], name, value)
		case fieldArg:
			data[name] = value
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
	return
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
