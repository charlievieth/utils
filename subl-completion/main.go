package main

import (
	"github.com/posener/complete/v2"
	"github.com/posener/complete/v2/predict"
)

func main() {
	subl := &complete.Command{
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
