package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const DefaultTimestamp = "15:04:05.000000"

var (
	DefaultStartDateUnix     = AddYears(time.Now(), -10).Unix()
	DefaultStartDateUnixNano = AddYears(time.Now(), -10).UnixNano()
	DefaultEndDateUnix       = AddYears(time.Now(), 10).Unix()
	DefaultEndDateUnixNano   = AddYears(time.Now(), 10).UnixNano()
)

func AddYears(t time.Time, years int) time.Time {
	return time.Date(t.Year()+years, t.Month(), t.Day(), t.Hour(),
		t.Minute(), t.Second(), 0, t.Location())
}

type StringArrayValue []string

func (a *StringArrayValue) Set(s string) error {
	*a = append(*a, s)
	return nil
}

func (a StringArrayValue) String() string {
	return fmt.Sprintf("%q", []string(a))
}

type Pattern struct {
	raw         string
	expr        string
	re          *regexp.Regexp
	subexpCount int
}

var replaceRe = regexp.MustCompile(`[^{]]*\{\}[^}]]*`)

func (p *Pattern) Compile(ignoreCase bool) error {
	if p.re != nil {
		return errors.New("pattern already compiled")
	}
	p.expr = replaceRe.ReplaceAllStringFunc(p.raw, func(s string) string {
		p.subexpCount++
		return fmt.Sprintf(`(?P<EXP_%d>\d+(?:\.\d+)?)`, p.subexpCount-1)
	})
	if ignoreCase {
		p.expr = `(?mi)` + p.expr
	} else {
		p.expr = `(?m)` + p.expr
	}
	re, err := regexp.Compile(p.expr)
	if err != nil {
		return err
	}
	p.re = re
	return nil
}

var rootCmd = &cobra.Command{
	Use: "gounix",
}

// var (
// 	durationRe         = regexp.MustCompile(`^(-?\d+(\.\d+)?)(ns|us|µs|μs|ms|s|m|h|D|M|Y)$`)
// 	extendedDurationRe = regexp.MustCompile(`^(-?\d+(\.\d+)?)(D|M|Y)$`)
// )

const (
	Day   = time.Hour * 24
	Month = Day * 30 // not precise
	Year  = Day * 365
)

// type extendedDuration time.Duration

// func (d *extendedDuration) Set(s string) error {
// 	if a := extendedDurationRe.FindStringSubmatch(s); len(a) == 3 {
// 		n, err := strconv.ParseInt(s, 10, 64)
// 	}
// 	v, err := time.ParseDuration(s)
// 	if err != nil {
// 		return err
// 	}
// 	*d = extendedDuration(v)
//
// 	return nil
// }

/*
// -- time.Duration Value
type durationValue time.Duration

func newDurationValue(val time.Duration, p *time.Duration) *durationValue {
	*p = val
	return (*durationValue)(p)
}

func (d *durationValue) Set(s string) error {
	v, err := time.ParseDuration(s)
	*d = durationValue(v)
	return err
}

func (d *durationValue) Type() string {
	return "duration"
}

func (d *durationValue) String() string { return (*time.Duration)(d).String() }

func durationConv(sval string) (interface{}, error) {
	return time.ParseDuration(sval)
}

// GetDuration return the duration value of a flag with the given name
func (f *FlagSet) GetDuration(name string) (time.Duration, error) {
	val, err := f.getFlagType(name, "duration", durationConv)
	if err != nil {
		return 0, err
	}
	return val.(time.Duration), nil
}
*/

func captureRegex(before, after int64) string {
	n1 := len(strconv.FormatInt(before, 10))
	n2 := len(strconv.FormatInt(after, 10))
	if n1 == n2 {
		return fmt.Sprintf(`[[:digit:]]{%d}`, n1)
	}
	if n1 < n2 {
		return fmt.Sprintf(`[[:digit:]]{%d,%d}`, n1, n2)
	}
	return fmt.Sprintf(`[[:digit:]]{%d,%d}`, n2, n1)
}

type Config struct {
	Before, After int64
	Format        string
	Re            *regexp.Regexp
	Nano          bool
}

func (c *Config) replaceInt(s string) string {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return s // WARN: log
	}
	if n < c.After || n > c.Before {
		return s
	}
	if c.Nano {
		return time.Unix(0, n).Format(c.Format)
	}
	return time.Unix(n, 0).Format(c.Format)
}

func (c *Config) replace(s string) string {
	if strings.IndexByte(s, '.') == -1 {
		return c.replaceInt(s)
	}
	panic("float not supported")
}

func (c *Config) Process(f *os.File) error {
	r := bufio.NewReaderSize(f, 16*1024)
	var err error
	for {
		b, e := r.ReadBytes('\n')
		if len(b) != 0 {
			_, we := os.Stdout.WriteString(c.Re.ReplaceAllStringFunc(string(b), c.replace))
			if we != nil {
				if e == nil || e == io.EOF {
					e = we
				}
			}
		}
		if e != nil {
			if e != io.EOF {
				err = e
			}
			break
		}
	}
	return err
}

func Command() *cobra.Command {
	// "ns"
	// "us"
	// "µs"
	// "μs"
	// "ms"
	// "s"
	// "m"
	// "h"

	root := &cobra.Command{
		Use: "gounix",

		RunE: func(cmd *cobra.Command, args []string) error {
			ad, err := cmd.Flags().GetDuration("after")
			if err != nil {
				return err
			}
			bd, err := cmd.Flags().GetDuration("before")
			if err != nil {
				return err
			}
			fmt, err := cmd.Flags().GetString("output-format")
			if err != nil {
				return err
			}
			nano, err := cmd.Flags().GetBool("nano")
			if err != nil {
				return err
			}
			now := time.Now()
			conf := Config{
				After:  now.Add(-ad).Unix(),
				Before: now.Add(bd).Unix(),
				Re:     regexp.MustCompile(`\d+`),
				Format: fmt,
				Nano:   nano,
			}
			return conf.Process(os.Stdin)

			// useStdin := len(args) == 0 || (len(args) == 1 && args[0] == "-")
			// _ = useStdin
			// return nil
		},
	}
	flags := root.Flags()
	// flags.Duration("after", value, usage)

	// TODO:
	//  1. note that `{}` is replaced
	//  2. support user supplied regexes like `(\d)\.foo` so long as they
	//     have one match group
	flags.StringSlice("regex", nil, "pattern to match against (ex: `{}\\.log$`) "+
		"may be specified multiple times")

	flags.Duration("after", Year, "only match timestamps after now() - this delta")
	flags.MarkHidden("after")

	flags.Duration("before", Year, "only match timestamps before now() + this delta")
	flags.MarkHidden("before")

	flags.Bool("nano", false, "parse timestamps as Unix nano")

	flags.StringP("output-format", "f", time.RFC3339, "format of converted unix timestamps")
	return root
}

func main() {
	if err := Command().Execute(); err != nil {
		Fatal(err)
	}
	return

	const raw = `\s+{}.hist \s+{}.hist$`
	n := 0
	s := replaceRe.ReplaceAllStringFunc(raw, func(s string) string {
		// (?P<first>[a-zA-Z]+)
		// (?P<EXP_1>\d+(?:\.\d+)?)

		out := fmt.Sprintf(`(?P<EXP_%d>\d+(?:\.\d+)?)`, n)
		n++
		return out
	})
	fmt.Println(s)
}

func oldMain() {
	var parseDur bool
	flag.BoolVar(&parseDur, "dur", false, "Calculate time since [TIMESTAMP]")
	flag.BoolVar(&parseDur, "d", false, "Calculate time since [TIMESTAMP] (shorthand)")
	flag.Parse()

	if parseDur {
		if flag.NArg() != 1 {
			Fatal(fmt.Sprintf("USAGE: %s -dur TIMESTAMP", filepath.Base(os.Args[0])))
		}
		// this is lazy, but whatever
		now, err := time.Parse(DefaultTimestamp, time.Now().Format(DefaultTimestamp))
		if err != nil {
			Fatal(err) // this should never happen
		}
		t, err := time.Parse(DefaultTimestamp, flag.Arg(0))
		if err != nil {
			Fatal(err)
		}
		fmt.Println(now.Sub(t).String())
		return
	}

	fmt.Println(time.Now().Format(DefaultTimestamp))
}

func Fatal(err interface{}) {
	if err == nil {
		return
	}
	var s string
	if _, file, line, ok := runtime.Caller(1); ok && file != "" {
		s = fmt.Sprintf("Error (%s:%d)", filepath.Base(file), line)
	} else {
		s = "Error"
	}
	switch err.(type) {
	case error, string, fmt.Stringer:
		fmt.Fprintf(os.Stderr, "%s: %s\n", s, err)
	default:
		fmt.Fprintf(os.Stderr, "%s: %#v\n", s, err)
	}
	os.Exit(1)
}
