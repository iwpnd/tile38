package server

import (
	"bytes"
	"strings"
	"time"

	"github.com/tidwall/resp"
	"github.com/tidwall/tile38/internal/glob"
)

func (s *Server) cmdKeys(msg *Message) (res resp.Value, err error) {
	var start = time.Now()
	vs := msg.Args[1:]

	var pattern string
	var ok bool
	if vs, pattern, ok = tokenval(vs); !ok || pattern == "" {
		return NOMessage, errInvalidNumberOfArguments
	}
	if len(vs) != 0 {
		return NOMessage, errInvalidNumberOfArguments
	}

	var wr = &bytes.Buffer{}
	var once bool
	if msg.OutputType == JSON {
		wr.WriteString(`{"ok":true,"keys":[`)
	}
	var wild bool
	if strings.Contains(pattern, "*") {
		wild = true
	}
	var everything bool
	var greater bool
	var greaterPivot string
	var vals []resp.Value

	iterator := func(v interface{}) bool {
		vcol := v.(*collectionKeyContainer)
		var match bool
		if everything {
			match = true
		} else if greater {
			if !strings.HasPrefix(vcol.key, greaterPivot) {
				return false
			}
			match = true
		} else {
			match, _ = glob.Match(pattern, vcol.key)
		}
		if match {
			if once {
				if msg.OutputType == JSON {
					wr.WriteByte(',')
				}
			} else {
				once = true
			}
			switch msg.OutputType {
			case JSON:
				wr.WriteString(jsonString(vcol.key))
			case RESP:
				vals = append(vals, resp.StringValue(vcol.key))
			}

			// If no more than one match is expected, stop searching
			if !wild {
				return false
			}
		}
		return true
	}

	// TODO: This can be further optimized by using glob.Parse and limits
	if pattern == "*" {
		everything = true
		s.cols.Ascend(nil, iterator)
	} else if strings.HasSuffix(pattern, "*") {
		greaterPivot = pattern[:len(pattern)-1]
		if glob.IsGlob(greaterPivot) {
			s.cols.Ascend(nil, iterator)
		} else {
			greater = true
			s.cols.Ascend(&collectionKeyContainer{key: greaterPivot}, iterator)
		}
	} else {
		s.cols.Ascend(nil, iterator)
	}
	if msg.OutputType == JSON {
		wr.WriteString(`],"elapsed":"` + time.Since(start).String() + "\"}")
		return resp.StringValue(wr.String()), nil
	}
	return resp.ArrayValue(vals), nil
}
