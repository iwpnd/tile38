package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/resp"
)

// Client is an remote connection into to Tile38
type Client struct {
	id         int            // unique id
	replPort   int            // the known replication port for follower connections
	authd      bool           // client has been authenticated
	outputType Type           // Null, JSON, or RESP
	remoteAddr string         // original remote address
	in         InputStream    // input stream
	pr         PipelineReader // command reader
	out        []byte         // output write buffer

	goLiveErr error    // error type used for going line
	goLiveMsg *Message // last message for go live

	mu     sync.Mutex         // guard
	conn   io.ReadWriteCloser // out-of-loop connection.
	name   string             // optional defined name
	opened time.Time          // when the client was created/opened, unix nano
	last   time.Time          // last client request/response, unix nano
}

// Write ...
func (client *Client) Write(b []byte) (n int, err error) {
	client.out = append(client.out, b...)
	return len(b), nil
}

type byID []*Client

func (arr byID) Len() int {
	return len(arr)
}
func (arr byID) Less(a, b int) bool {
	return arr[a].id < arr[b].id
}
func (arr byID) Swap(a, b int) {
	arr[a], arr[b] = arr[b], arr[a]
}

func (s *Server) cmdClient(msg *Message, client *Client) (resp.Value, error) {
	start := time.Now()

	if len(msg.Args) == 1 {
		return NOMessage, errInvalidNumberOfArguments
	}
	switch strings.ToLower(msg.Args[1]) {
	default:
		return NOMessage, clientErrorf(
			"Syntax error, try CLIENT (LIST | KILL | GETNAME | SETNAME)",
		)
	case "list":
		if len(msg.Args) != 2 {
			return NOMessage, errInvalidNumberOfArguments
		}
		var list []*Client
		s.connsmu.RLock()
		for _, cc := range s.conns {
			list = append(list, cc)
		}
		s.connsmu.RUnlock()
		sort.Sort(byID(list))
		now := time.Now()
		var buf []byte
		for _, client := range list {
			client.mu.Lock()
			buf = append(buf,
				fmt.Sprintf("id=%d addr=%s name=%s age=%d idle=%d\n",
					client.id,
					client.remoteAddr,
					client.name,
					now.Sub(client.opened)/time.Second,
					now.Sub(client.last)/time.Second,
				)...,
			)
			client.mu.Unlock()
		}
		switch msg.OutputType {
		case JSON:
			// Create a map of all key/value info fields
			var cmap []map[string]interface{}
			clients := strings.Split(string(buf), "\n")
			for _, client := range clients {
				client = strings.TrimSpace(client)
				m := make(map[string]interface{})
				var hasFields bool
				for _, kv := range strings.Split(client, " ") {
					kv = strings.TrimSpace(kv)
					if split := strings.SplitN(kv, "=", 2); len(split) == 2 {
						hasFields = true
						m[split[0]] = tryParseType(split[1])
					}
				}
				if hasFields {
					cmap = append(cmap, m)
				}
			}

			// Marshal the map and use the output in the JSON response
			data, err := json.Marshal(cmap)
			if err != nil {
				return NOMessage, err
			}
			return resp.StringValue(`{"ok":true,"list":` + string(data) + `,"elapsed":"` + time.Since(start).String() + "\"}"), nil
		case RESP:
			return resp.BytesValue(buf), nil
		}
		return NOMessage, nil
	case "getname":
		if len(msg.Args) != 2 {
			return NOMessage, errInvalidNumberOfArguments
		}
		name := ""
		switch msg.OutputType {
		case JSON:
			client.mu.Lock()
			name := client.name
			client.mu.Unlock()
			return resp.StringValue(`{"ok":true,"name":` +
				jsonString(name) +
				`,"elapsed":"` + time.Since(start).String() + "\"}"), nil
		case RESP:
			return resp.StringValue(name), nil
		}
	case "setname":
		if len(msg.Args) != 3 {
			return NOMessage, errInvalidNumberOfArguments
		}
		name := msg.Args[2]
		for i := 0; i < len(name); i++ {
			if name[i] < '!' || name[i] > '~' {
				return NOMessage, clientErrorf(
					"Client names cannot contain spaces, newlines or special characters.",
				)
			}
		}
		client.mu.Lock()
		client.name = name
		client.mu.Unlock()
		switch msg.OutputType {
		case JSON:
			return resp.StringValue(`{"ok":true,"elapsed":"` + time.Since(start).String() + "\"}"), nil
		case RESP:
			return resp.SimpleStringValue("OK"), nil
		}
	case "kill":
		if len(msg.Args) < 3 {
			return NOMessage, errInvalidNumberOfArguments
		}
		var useAddr bool
		var addr string
		var useID bool
		var id string
		for i := 2; i < len(msg.Args); i++ {
			arg := msg.Args[i]
			if strings.Contains(arg, ":") {
				addr = arg
				useAddr = true
				break
			}
			switch strings.ToLower(arg) {
			default:
				return NOMessage, clientErrorf("No such client")
			case "addr":
				i++
				if i == len(msg.Args) {
					return NOMessage, errors.New("syntax error")
				}
				addr = msg.Args[i]
				useAddr = true
			case "id":
				i++
				if i == len(msg.Args) {
					return NOMessage, errors.New("syntax error")
				}
				id = msg.Args[i]
				useID = true
			}
		}
		var cclose *Client
		s.connsmu.RLock()
		for _, cc := range s.conns {
			if useID && fmt.Sprintf("%d", cc.id) == id {
				cclose = cc
				break
			} else if useAddr && client.remoteAddr == addr {
				cclose = cc
				break
			}
		}
		s.connsmu.RUnlock()
		if cclose == nil {
			return NOMessage, clientErrorf("No such client")
		}

		var res resp.Value
		switch msg.OutputType {
		case JSON:
			res = resp.StringValue(`{"ok":true,"elapsed":"` + time.Since(start).String() + "\"}")
		case RESP:
			res = resp.SimpleStringValue("OK")
		}

		client.conn.Close()
		// closing self, return response now
		// NOTE: This is the only exception where we do convert response to a string
		var outBytes []byte
		switch msg.OutputType {
		case JSON:
			outBytes = res.Bytes()
		case RESP:
			outBytes, _ = res.MarshalRESP()
		}
		cclose.conn.Write(outBytes)
		cclose.conn.Close()
		return res, nil
	}
	return NOMessage, errors.New("invalid output type")
}
