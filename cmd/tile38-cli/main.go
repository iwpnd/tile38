package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/peterh/liner"
	"github.com/tidwall/gjson"
	"github.com/tidwall/resp"
	"github.com/tidwall/tile38/core"
)

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

var (
	historyFile = filepath.Join(userHomeDir(), ".liner_example_history")
)

type connError struct {
	OK  bool   `json:"ok"`
	Err string `json:"err"`
}

var (
	hostname   = "127.0.0.1"
	output     = "json"
	port       = 9851
	oneCommand string
	raw        bool
	noprompt   bool
	tty        bool
)

func showHelp() bool {

	gitsha := ""
	if core.GitSHA == "" || core.GitSHA == "0000000" {
		gitsha = ""
	} else {
		gitsha = " (git:" + core.GitSHA + ")"
	}
	fmt.Fprintf(os.Stdout, "tile38-cli %s%s\n\n", core.Version, gitsha)
	fmt.Fprintf(os.Stdout, "Usage: tile38-cli [OPTIONS] [cmd [arg [arg ...]]]\n")
	fmt.Fprintf(os.Stdout, " --raw              Use raw formatting for replies\n")
	fmt.Fprintf(os.Stdout, " --noprompt         Do not display a prompt\n")
	fmt.Fprintf(os.Stdout, " --resp             Use RESP output formatting (default is JSON output)\n")
	fmt.Fprintf(os.Stdout, " --json             Use JSON output formatting (default is JSON output)\n")
	fmt.Fprintf(os.Stdout, " -h <hostname>      Server hostname (default: %s)\n", hostname)
	fmt.Fprintf(os.Stdout, " -p <port>          Server port (default: %d)\n", port)
	fmt.Fprintf(os.Stdout, "\n")
	return false
}

func parseArgs() bool {
	defer func() {
		if v := recover(); v != nil {
			if v, ok := v.(string); ok && v == "bad arg" {
				showHelp()
			}
		}
	}()

	args := os.Args[1:]
	readArg := func(arg string) string {
		if len(args) == 0 {
			panic("bad arg")
		}
		var narg = args[0]
		args = args[1:]
		return narg
	}
	badArg := func(arg string) bool {
		fmt.Fprintf(os.Stderr, "Unrecognized option or bad number of args for: '%s'\n", arg)
		return false
	}

	for len(args) > 0 {
		arg := readArg("")
		if arg == "--help" || arg == "-?" {
			return showHelp()
		}
		if !strings.HasPrefix(arg, "-") {
			args = append([]string{arg}, args...)
			break
		}
		switch arg {
		default:
			return badArg(arg)
		case "--raw":
			raw = true
		case "--tty":
			tty = true
		case "--noprompt":
			noprompt = true
		case "--resp":
			output = "resp"
		case "--json":
			output = "json"
		case "-h":
			hostname = readArg(arg)
		case "-p":
			n, err := strconv.ParseUint(readArg(arg), 10, 16)
			if err != nil {
				return badArg(arg)
			}
			port = int(n)
		}
	}
	oneCommand = strings.Join(args, " ")
	return true
}

func refusedErrorString(addr string) string {
	return fmt.Sprintf("Could not connect to Tile38 at %s: Connection refused", addr)
}

var groupsM = make(map[string][]string)

func jsonOK(msg []byte) bool {
	return gjson.GetBytes(msg, "ok").Bool()
}

func main() {
	if !parseArgs() {
		return
	}

	if len(oneCommand) > 0 && (oneCommand[0] == 'h' || oneCommand[0] == 'H') && strings.Split(strings.ToLower(oneCommand), " ")[0] == "help" {
		showHelp()
		return
	}

	addr := fmt.Sprintf("%s:%d", hostname, port)
	var conn *client
	connDial := func() {
		var err error
		conn, err = clientDial("tcp", addr)
		if err != nil {
			if _, ok := err.(net.Error); ok {
				fmt.Fprintln(os.Stderr, refusedErrorString(addr))
			} else {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			if oneCommand != "" {
				os.Exit(1)
			}
		} else if _, err := conn.Do("output " + output); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}
	connDial()
	monitor := false
	livemode := false
	aof := false
	defer func() {
		if livemode {
			var err error
			if aof {
				_, err = io.Copy(os.Stdout, conn.Reader())
				fmt.Fprintln(os.Stderr, "")
			} else {
				var msg []byte
				for {
					msg, err = conn.readLiveResp()
					if err != nil {
						break
					}
					if !raw {
						if output == "resp" {
							msg = convert2termresp(msg)
						} else {
							msg = convert2termjson(msg)
						}
					}
					fmt.Fprintln(os.Stderr, string(msg))
				}
			}
			if err != nil && err != io.EOF {
				fmt.Fprintln(os.Stderr, err.Error())
			}
		}
	}()

	line := liner.NewLiner()
	defer line.Close()

	var commands []string
	for name, command := range core.Commands {
		commands = append(commands, name)
		groupsM[command.Group] = append(groupsM[command.Group], name)
	}
	sort.Strings(commands)
	var groups []string
	for group, arr := range groupsM {
		groups = append(groups, "@"+group)
		sort.Strings(arr)
		groupsM[group] = arr
	}
	sort.Strings(groups)

	line.SetMultiLineMode(false)
	line.SetCtrlCAborts(true)
	if !(noprompt && tty) {
		line.SetCompleter(func(line string) (c []string) {
			if strings.HasPrefix(strings.ToLower(line), "help ") {
				var nitems []string
				nline := strings.TrimSpace(line[5:])
				if nline == "" || nline[0] == '@' {
					for _, n := range groups {
						if strings.HasPrefix(strings.ToLower(n), strings.ToLower(nline)) {
							nitems = append(nitems, line[:len(line)-len(nline)]+strings.ToLower(n))
						}
					}
				} else {
					for _, n := range commands {
						if strings.HasPrefix(strings.ToLower(n), strings.ToLower(nline)) {
							nitems = append(nitems, line[:len(line)-len(nline)]+strings.ToUpper(n))
						}
					}
				}
				for _, n := range nitems {
					if strings.HasPrefix(strings.ToLower(n), strings.ToLower(line)) {
						c = append(c, n)
					}
				}
			} else {
				for _, n := range commands {
					if strings.HasPrefix(strings.ToLower(n), strings.ToLower(line)) {
						c = append(c, n)
					}
				}
			}
			return
		})
	}
	if f, err := os.Open(historyFile); err == nil {
		line.ReadHistory(f)
		f.Close()
	}
	defer func() {
		if f, err := os.Create(historyFile); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		} else {
			line.WriteHistory(f)
			f.Close()
		}
	}()
	for {
		var command string
		var err error
		if oneCommand == "" {
			if raw || noprompt {
				command, err = line.Prompt("")
			} else {
				if conn == nil {
					command, err = line.Prompt("not connected> ")
				} else {
					command, err = line.Prompt(addr + "> ")
				}
			}

		} else {
			command = oneCommand
		}
		if err == nil {
			nohist := strings.HasPrefix(command, " ")
			command = strings.TrimSpace(command)
			if command == "" {
				if conn != nil {
					_, err := conn.Do("pInG")
					if err != nil {
						if err != io.EOF && !strings.Contains(err.Error(), "broken pipe") {
							fmt.Fprintln(os.Stderr, err.Error())
						} else {
							fmt.Fprintln(os.Stderr, refusedErrorString(addr))
						}
						conn.wr.Close()
						conn = nil
						continue
					}
				}
			} else {
				if !nohist {
					line.AppendHistory(command)
				}
				if (command[0] == 'e' || command[0] == 'E') && strings.ToLower(command) == "exit" {
					return
				}
				if (command[0] == 'q' || command[0] == 'Q') && strings.ToLower(command) == "quit" {
					return
				}
				if (command[0] == 'h' || command[0] == 'H') && (strings.ToLower(command) == "help" || strings.HasPrefix(strings.ToLower(command), "help")) {
					err = help(strings.TrimSpace(command[4:]))
					if err != nil {
						return
					}
					continue
				}
				aof = (command[0] == 'a' || command[0] == 'A') && strings.HasPrefix(strings.ToLower(command), "aof ")
			tryAgain:
				if conn == nil {
					connDial()
					if conn == nil {
						continue
					}
				}
				msg, err := conn.Do(command)
				if err != nil {
					if err != io.EOF {
						fmt.Fprintln(os.Stderr, err.Error())
					}
					conn.wr.Close()
					conn = nil
					goto tryAgain
				}
				switch strings.ToLower(command) {
				case "output resp":
					if string(msg) == "+OK\r\n" {
						output = "resp"
					}
				case "output json":
					if jsonOK(msg) {
						output = "json"
					}
				case "monitor":
					monitor = true
					livemode = true
					output = "resp"
				}
				if output == "resp" &&
					(strings.HasPrefix(string(msg), "*3\r\n$10\r\npsubscribe\r\n") ||
						strings.HasPrefix(string(msg), "*3\r\n$9\r\nsubscribe\r\n")) {
					livemode = true
				}
				if !raw {
					if output == "resp" {
						msg = convert2termresp(msg)
					} else {
						msg = convert2termjson(msg)
					}
				}

				if !livemode && output == "json" {
					if gjson.GetBytes(msg, "command").String() == "psubscribe" ||
						gjson.GetBytes(msg, "command").String() == "subscribe" ||
						string(msg) == liveJSON {
						livemode = true
					}
				}

				mustOutput := true
				if !monitor && oneCommand == "" && output == "json" && !jsonOK(msg) {
					var cerr connError
					if err := json.Unmarshal(msg, &cerr); err == nil {
						fmt.Fprintln(os.Stderr, "(error) "+cerr.Err)
						mustOutput = false
					}
				} else if livemode {
					fmt.Fprintln(os.Stderr, string(msg))
					break // break out of prompt and just feed data to screen
				}
				if mustOutput {
					fmt.Fprintln(os.Stdout, string(msg))
				}
			}
		} else if err == liner.ErrPromptAborted {
			return
		} else if err == io.EOF {
			os.Exit(0)
		} else {
			fmt.Fprintf(os.Stderr, "Error reading line: %s", err.Error())
		}
		if oneCommand != "" {
			return
		}
	}
}

func convert2termresp(msg []byte) []byte {
	rd := resp.NewReader(bytes.NewBuffer(msg))
	out := ""
	for {
		v, _, err := rd.ReadValue()
		if err != nil {
			break
		}
		out += convert2termrespval(v, 0)
	}
	return []byte(strings.TrimSpace(out))
}

func convert2termjson(msg []byte) []byte {
	if msg[0] == '{' {
		return msg
	}
	return bytes.TrimSpace(msg[bytes.IndexByte(msg, '\n')+1:])
}

func convert2termrespval(v resp.Value, spaces int) string {
	switch v.Type() {
	default:
		return v.String()
	case resp.BulkString:
		if v.IsNull() {
			return "(nil)"
		}
		return "\"" + v.String() + "\""
	case resp.Integer:
		return "(integer) " + v.String()
	case resp.Error:
		return "(error) " + v.String()
	case resp.Array:
		arr := v.Array()
		if len(arr) == 0 {
			return "(empty list or set)"
		}
		out := ""
		nspaces := spaces + numlen(len(arr))
		for i, v := range arr {
			if i > 0 {
				out += strings.Repeat(" ", spaces)
			}
			iout := strings.TrimSpace(convert2termrespval(v, nspaces+2))
			out += fmt.Sprintf("%d) %s\n", i+1, iout)
		}
		return out
	}
}

func numlen(n int) int {
	l := 1
	if n < 0 {
		l++
		n = n * -1
	}
	for i := 0; i < 1000; i++ {
		if n < 10 {
			break
		}
		l++
		n = n / 10
	}
	return l
}

func help(arg string) error {
	var groupsA []string
	for group := range groupsM {
		groupsA = append(groupsA, "@"+group)
	}
	groups := "Groups: " + strings.Join(groupsA, ", ") + "\n"

	if arg == "" {
		fmt.Fprintf(os.Stderr, "tile38-cli %s (git:%s)\n", core.Version, core.GitSHA)
		fmt.Fprintf(os.Stderr, `Type:   "help @<group>" to get a list of commands in <group>`+"\n")
		fmt.Fprintf(os.Stderr, `        "help <command>" for help on <command>`+"\n")
		if !(noprompt && tty) {
			fmt.Fprintf(os.Stderr, `        "help <tab>" to get a list of possible help topics`+"\n")
		}
		fmt.Fprintf(os.Stderr, `        "quit" to exit`+"\n")
		if noprompt && tty {
			fmt.Fprint(os.Stderr, groups)
		}
		return nil
	}
	showGroups := false
	found := false
	if strings.HasPrefix(arg, "@") {
		for _, command := range groupsM[arg[1:]] {
			fmt.Fprintf(os.Stderr, "%s\n", core.Commands[command].TermOutput("  "))
			found = true
		}
		if !found {
			showGroups = true
		}
	} else {
		if command, ok := core.Commands[strings.ToUpper(arg)]; ok {
			fmt.Fprintf(os.Stderr, "%s\n", command.TermOutput("  "))
			found = true
		}
	}
	if showGroups {
		if noprompt && tty {
			fmt.Fprint(os.Stderr, groups)
		}
	} else if !found {
		if noprompt && tty {
			help("")
		}
	}
	return nil
}

const liveJSON = `{"ok":true,"live":true}`

type client struct {
	wr net.Conn
	rd *bufio.Reader
}

func clientDial(network, addr string) (*client, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	return &client{wr: conn, rd: bufio.NewReader(conn)}, nil
}

func (c *client) Do(command string) ([]byte, error) {
	_, err := c.wr.Write(plainToCompat(command))
	if err != nil {
		return nil, err
	}
	return c.readResp()
}

func (c *client) readResp() ([]byte, error) {
	ch, err := c.rd.Peek(1)
	if err != nil {
		return nil, err
	}
	switch ch[0] {
	case ':', '+', '-', '{':
		return c.readLine()
	case '$':
		return c.readBulk()
	case '*':
		return c.readArray()
	default:
		return nil, fmt.Errorf("invalid response character '%c", ch[0])
	}
}

func (c *client) readArray() ([]byte, error) {
	out, err := c.readLine()
	if err != nil {
		return nil, err
	}
	n, err := strconv.ParseUint(string(bytes.TrimSpace(out[1:])), 10, 64)
	if err != nil {
		return nil, err
	}
	for i := 0; i < int(n); i++ {
		resp, err := c.readResp()
		if err != nil {
			return nil, err
		}
		out = append(out, resp...)
	}
	return out, nil
}

func (c *client) readBulk() ([]byte, error) {
	line, err := c.readLine()
	if err != nil {
		return nil, err
	}
	x, err := strconv.ParseInt(string(bytes.TrimSpace(line[1:])), 10, 64)
	if err != nil {
		return nil, err
	}
	if x < 0 {
		return line, nil
	}
	out := make([]byte, len(line)+int(x)+2)
	if _, err := io.ReadFull(c.rd, out[len(line):]); err != nil {
		return nil, err
	}
	if !bytes.HasSuffix(out, []byte{'\r', '\n'}) {
		return nil, errors.New("invalid response")
	}
	copy(out, line)
	return out, nil
}

func (c *client) readLine() ([]byte, error) {
	line, err := c.rd.ReadBytes('\r')
	if err != nil {
		return nil, err
	}
	ch, err := c.rd.ReadByte()
	if err != nil {
		return nil, err
	}
	if ch != '\n' {
		return nil, errors.New("invalid response")
	}
	return append(line, '\n'), nil
}

func (c *client) Reader() io.Reader {
	return c.rd
}

func (c *client) readLiveResp() (message []byte, err error) {
	return c.readResp()
}

// plainToCompat converts a plain message like "SET fleet truck1 ..."  into a
// Tile38 compatible blob.
func plainToCompat(message string) []byte {
	var args []string
	// search for the beginning of the first argument
	for i := 0; i < len(message); i++ {
		if message[i] != ' ' {
			// first argument found
			if message[i] == '"' || message[i] == '\'' {
				// using a string caps
				s := i
				cap := message[i]
				for ; i < len(message); i++ {
					if message[i] == cap {
						if message[i-1] == '\\' {
							continue
						}
						if i == len(message)-1 || message[i+1] == ' ' {
							args = append(args, message[s:i+1])
							i++
							break
						}
					}
				}
			} else {
				// using plain string, terminated by a space
				s := i
				var quotes bool
				for ; i < len(message); i++ {
					if message[i] == '"' || message[i] == '\'' {
						quotes = true
					}
					if i == len(message)-1 || message[i+1] == ' ' {
						arg := message[s : i+1]
						if quotes {
							arg = strconv.Quote(arg)
						}
						args = append(args, arg)
						i++
						break
					}
				}
			}
		}
	}
	return []byte(strings.Join(args, " ") + "\r\n")
}
