package main

import (
	"bytes"
	"os"
	"sort"
	"time"

	"github.com/posener/complete/v2"
	"github.com/posener/complete/v2/predict"
)

func isBinary(buf []byte, name string) bool {
	f, err := os.Open(name)
	if err != nil {
		return true // ignore
	}
	n, err := f.Read(buf)
	f.Close()
	if err != nil {
		return true // ignore
	}
	return bytes.IndexByte(buf[:n], 0) != -1
}

func Complete(prefix string) (options []string) {
	dirs := make(chan []string, 1)
	go func() { dirs <- predict.Dirs("*").Predict(prefix) }()

	buf := make([]byte, 512)
	names := predict.Files("*").Predict(prefix)
	a := names[:0]
	for _, name := range names {
		if !isBinary(buf, name) {
			a = append(a, name)
		}
	}
	names = a

	select {
	case d := <-dirs:
		names = append(names, d...)
		sort.Strings(names)
	case <-time.After(time.Millisecond * 200):
		// ok
	}
	return names
}

func main() {
	subl := &complete.Command{
		Args: complete.PredictFunc(Complete),
		Flags: map[string]complete.Predictor{
			"a ":                   predict.Nothing,
			"add":                  predict.Nothing,
			"w ":                   predict.Nothing,
			"wait":                 predict.Nothing,
			"b ":                   predict.Nothing,
			"background":           predict.Nothing,
			"s ":                   predict.Nothing,
			"stay":                 predict.Nothing,
			"safe-mode":            predict.Nothing,
			"h ":                   predict.Nothing,
			"help":                 predict.Nothing,
			"v ":                   predict.Nothing,
			"version":              predict.Nothing,
			"n ":                   predict.Nothing,
			"new-window":           predict.Nothing,
			"command":              predict.Nothing,
			"launch-or-new-window": predict.Nothing,
			"project":              predict.Files("*.sublime-project"),
		},
	}
	subl.Complete("subl")
}
