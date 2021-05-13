package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/posener/complete/v2"
	"github.com/posener/complete/v2/predict"
)

/*
Usage: watchman [opts] command

 -h, --help                 Show this help

     --inetd                Spawning from an inetd style supervisor

 -v, --version              Show version number

 -U, --sockname=PATH        Specify alternate sockname

 -o, --logfile=PATH         Specify path to logfile

     --log-level            set the log level (0 = off, default is 1, verbose = 2)

     --pidfile=PATH         Specify path to pidfile

 -p, --persistent           Persist and wait for further responses

 -n, --no-save-state        Don't save state between invocations

     --statefile=PATH       Specify path to file to hold watch and trigger state

 -j, --json-command         Instead of parsing CLI arguments, take a single json object from stdin

     --output-encoding=ARG  CLI output encoding. json (default) or bser

     --server-encoding=ARG  CLI<->server encoding. bser (default) or json

 -f, --foreground           Run the service in the foreground

     --no-pretty            Don't pretty print JSON

     --no-spawn             Don't try to start the service if it is not available

     --no-local             When no-spawn is enabled, don't try to handle request in client mode if service is unavailable


Available commands:

      clock
      debug-ageout
      debug-contenthash
      debug-drop-privs
      debug-fsevents-inject-drop
      debug-get-subscriptions
      debug-poison
      debug-recrawl
      debug-set-subscriptions-paused
      debug-show-cursors
      find
      flush-subscriptions
      get-config
      get-pid
      get-sockname
      list-capabilities
      log
      log-level
      query
      shutdown-server
      since
      state-enter
      state-leave
      subscribe
      trigger
      trigger-del
      trigger-list
      unsubscribe
      version
      watch
      watch-del
      watch-del-all
      watch-list
      watch-project

See https://github.com/facebook/watchman#watchman for more help

Watchman, by Wez Furlong.
Copyright 2012-2017 Facebook, Inc.
*/

/*
{
    "version": "4.9.0",
    "roots": [
        "/Users/cvieth/go/src/github.com/posener/complete",
        "/Users/cvieth/go/src/github.com/cockroachlabs/managed-service"
    ]
}
*/

type ListResponse struct {
	Version string   `json:"version"`
	Roots   []string `json:"roots"`
}

func WatchList() []string {
	out, err := exec.Command("watchman", "watch-list").Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "Error: %s\n%s\n",
				e.Error(), string(bytes.TrimSpace(e.Stderr)))
			os.Exit(1)
		}
	}
	var res ListResponse
	if err := json.Unmarshal(out, &res); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	return res.Roots
}

const CaseInsensitive = runtime.GOOS == "darwin" || runtime.GOOS == "windows"

func HasPathPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && (s[0:len(prefix)] == prefix ||
		(CaseInsensitive && strings.EqualFold(s[0:len(prefix)], prefix)))
}

func CompleteWatchList(prefix string) []string {
	watches := WatchList()
	if len(watches) == 0 {
		return nil
	}
	a := watches[:0]
	for _, s := range watches {
		if HasPathPrefix(s, prefix) {
			a = append(a, s)
		}
	}
	return a
}

func main() {
	watch := &complete.Command{
		Flags: map[string]complete.Predictor{
			"inetd":           predict.Nothing,
			"log-level":       predict.Nothing,
			"no-local":        predict.Nothing,
			"no-pretty":       predict.Nothing,
			"no-spawn":        predict.Nothing,
			"output-encoding": predict.Set{"json", "bser"},
			"pidfile":         predict.Files("*"),
			"server-encoding": predict.Set{"bser", "json"},
			"statefile":       predict.Files("*"),
			"f":               predict.Nothing,
			"foreground":      predict.Nothing,
			"h":               predict.Nothing,
			"help":            predict.Nothing,
			"j":               predict.Nothing,
			"json-command":    predict.Nothing,
			"n":               predict.Nothing,
			"no-save-state":   predict.Nothing,
			"o":               predict.Nothing,
			"logfile":         predict.Files("*"),
			"p":               predict.Nothing,
			"persistent":      predict.Nothing,
			"U":               predict.Nothing,
			"sockname":        predict.Files("*"),
			"v":               predict.Nothing,
			"version":         predict.Nothing,
		},
		Sub: map[string]*complete.Command{
			"clock":                          {},
			"debug-ageout":                   {},
			"debug-contenthash":              {},
			"debug-drop-privs":               {},
			"debug-fsevents-inject-drop":     {},
			"debug-get-subscriptions":        {},
			"debug-poison":                   {},
			"debug-recrawl":                  {},
			"debug-set-subscriptions-paused": {},
			"debug-show-cursors":             {},
			"find":                           {},
			"flush-subscriptions":            {},
			"get-config":                     {},
			"get-pid":                        {},
			"get-sockname":                   {},
			"list-capabilities":              {},
			"log":                            {},
			"log-level":                      {},
			"query":                          {},
			"shutdown-server": {
				Args: predict.Nothing,
			},
			"since":       {},
			"state-enter": {},
			"state-leave": {},
			"subscribe":   {},
			"trigger": {
				Args: complete.PredictFunc(CompleteWatchList),
			},
			"trigger-del": {},
			"trigger-list": {
				Args: complete.PredictFunc(CompleteWatchList),
			},
			"unsubscribe": {
				Args: complete.PredictFunc(CompleteWatchList),
			},
			"version": {},
			"watch": {
				Args: predict.Or(predict.Dirs("*"), predict.Files("*")),
			},
			"watch-del": {
				Args: complete.PredictFunc(CompleteWatchList),
			},
			"watch-del-all": {},
			"watch-list": {
				Args: complete.PredictFunc(CompleteWatchList),
			},
			"watch-project": {
				Args: predict.Or(predict.Dirs("*"), predict.Files("*")),
			},
		},
	}
	watch.Complete("watchman")
}
