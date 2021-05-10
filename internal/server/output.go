package server

import (
	"strings"
	"time"

	"github.com/tidwall/resp"
)

func (s *Server) cmdOutput(msg *Message) (res resp.Value, err error) {
	start := time.Now()
	vs := msg.Args[1:]
	var arg string
	var ok bool

	if len(vs) != 0 {
		if _, arg, ok = tokenval(vs); !ok || arg == "" {
			return NOMessage, errInvalidNumberOfArguments
		}
		// Setting the original message output type will be picked up by the
		// server prior to the next command being executed.
		switch strings.ToLower(arg) {
		default:
			return NOMessage, errInvalidArgument(arg)
		case "json":
			msg.OutputType = JSON
		case "resp":
			msg.OutputType = RESP
		}
		return OKMessage(msg, start), nil
	}
	// return the output
	switch msg.OutputType {
	default:
		return NOMessage, nil
	case JSON:
		return resp.StringValue(`{"ok":true,"output":"json","elapsed":` + time.Since(start).String() + `}`), nil
	case RESP:
		return resp.StringValue("resp"), nil
	}
}
