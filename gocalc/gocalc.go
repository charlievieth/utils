package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

const Precision = 200

type SuffixError struct {
	Suffix string
}

func (e *SuffixError) Error() string {
	return `invalid suffix: "` + e.Suffix + `"`
}

// TODO: support "MGTPEZY"
func suffixValue(suffix string) (uint64, error) {
	const (
		_  = iota
		kB = 1 << (10 * iota)
		mB // 2
		gB // 3
		tB // 4
		pB // 5
		eB // 6
		zB
	)
	switch len(suffix) {
	case 0:
		return 1, nil
	case 1:
		c := suffix[0]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		switch c {
		case 'b':
			return 1, nil
		case 'k':
			return kB, nil
		case 'm':
			return mB, nil
		case 'g':
			return gB, nil
		case 't':
			return tB, nil
		case 'p':
			return pB, nil
		case 'e':
			return eB, nil
		}
	case 2:
		switch strings.ToLower(suffix) {
		case "kb":
			return kB, nil
		case "mb":
			return mB, nil
		case "gb":
			return gB, nil
		case "tb":
			return tB, nil
		case "pb":
			return pB, nil
		case "eb":
			return eB, nil
		}
	}
	return 1, &SuffixError{Suffix: suffix}
}

func Suffix(s string) (val string, mult uint64, err error) {
	var sfx string
	for i := len(s) - 1; i >= 0; i-- {
		c := s[i]
		if '0' <= c && c <= '9' || c == '.' {
			val, sfx = s[:i+1], s[i+1:]
			break
		}
	}
	mult = 1
	if len(sfx) != 0 {
		mult, err = suffixValue(sfx)
	}
	return val, mult, err
}

func ParseValue(val string, useSuffix, parseDuration bool) (*big.Float, error) {
	if parseDuration {
		d, err := time.ParseDuration(val)
		if err != nil {
			return nil, err
		}
		var f big.Float
		return f.SetInt64(int64(d)), nil
	}
	mult := uint64(1)
	if useSuffix {
		var err error
		if val, mult, err = Suffix(val); err != nil {
			return nil, err
		}
	}
	f, _, err := big.ParseFloat(val, 10, Precision, 0)
	if err != nil {
		return nil, err
	}
	if mult != 1 {
		var m big.Float
		m.SetUint64(mult)
		f.Mul(f, &m)
	}
	return f, nil
}

type Percentile struct {
	P     float64    `json:"p"`
	Value *big.Float `json:"value"`
}

type Stats struct {
	PrintDuration bool          `json:"-"`
	Count         int64         `json:"count"`
	Sum           *big.Float    `json:"sum"`
	Average       *big.Float    `json:"average"`
	Median        *big.Float    `json:"median,omitempty"`
	Min           *big.Float    `json:"min,omitempty"`
	Max           *big.Float    `json:"max,omitempty"`
	Percentiles   []*Percentile `json:"percentiles,omitempty"`
}

func newStats(count int64, sum *big.Float, all []*big.Float, percentiles []float64) *Stats {
	var c big.Float
	c.SetInt64(count)
	stats := &Stats{
		Count:   count,
		Sum:     new(big.Float).Copy(sum),
		Average: new(big.Float).Quo(sum, &c),
	}
	if len(all) != 0 {
		sort.Slice(all, func(i, j int) bool {
			return all[i].Cmp(all[j]) == -1
		})
		stats.Median = all[len(all)/2]
		stats.Min = all[0]
		stats.Max = all[len(all)-1]
	}

	percentiles = uniqueFloats(percentiles)
	for _, p := range percentiles {
		k := (p / 100) * float64(len(all))
		i := int(math.Ceil(k))
		if i >= len(all) {
			i = len(all) - 1
		}
		stats.Percentiles = append(stats.Percentiles, &Percentile{
			P:     p,
			Value: all[i],
		})
	}
	return stats
}

func (s *Stats) formatValue(f *big.Float) string {
	if s.PrintDuration {
		i, _ := f.Int64()
		switch i {
		case math.MaxInt64, math.MinInt64:
			// Return the number of seconds on {over,under}flow
			return f.Text('f', 2) + "s"
		}
		return time.Duration(i).String()
	}
	if f.IsInt() {
		return f.Text('f', 0)
	}
	return f.Text('f', 2)
}

func (s *Stats) WriteTo(wr io.Writer) (int64, error) {
	var buf bytes.Buffer

	w := tabwriter.NewWriter(&buf, 1, 4, 2, ' ', 0)
	buf.WriteString("\nStats:\n")
	fmt.Fprintf(w, "  count:\t%d\n", s.Count)
	fmt.Fprintf(w, "  sum:\t%s\n", s.formatValue(s.Sum))
	fmt.Fprintf(w, "  average:\t%s\n", s.formatValue(s.Average))
	if s.Median != nil {
		fmt.Fprintf(w, "  median:\t%s\n", s.formatValue(s.Median))
	}
	if s.Min != nil {
		fmt.Fprintf(w, "  min:\t%s\n", s.formatValue(s.Min))
	}
	if s.Max != nil {
		fmt.Fprintf(w, "  max:\t%s\n", s.formatValue(s.Max))
	}
	w.Flush()

	if len(s.Percentiles) > 0 {
		w = w.Init(&buf, 1, 4, 2, ' ', 0)
		buf.WriteString("\nPercentiles:\n")
		for _, p := range s.Percentiles {
			fmt.Fprintf(w, "  %.0f:\t%s\n", p.P, s.formatValue(p.Value))
		}
		w.Flush()
	}

	return buf.WriteTo(wr)
}

func uniqueFloats(ff []float64) []float64 {
	if len(ff) <= 1 {
		return ff
	}
	sort.Float64s(ff)
	a := ff[:1]
	for i := 1; i < len(ff); i++ {
		if ff[i] != ff[i-1] {
			a = append(a, ff[i])
		}
	}
	return a
}

type Config struct {
	Median         bool
	Percentiles    []float64
	StdPercentiles bool
	P75            bool
	P90            bool
	P95            bool
	Human          bool
	Duration       bool
	JSON           bool
	JSONIndent     int
}

func (c *Config) GetPercentiles() []float64 {
	percentiles := append([]float64(nil), c.Percentiles...)
	if c.StdPercentiles {
		percentiles = append(percentiles, []float64{50, 75, 90, 95, 99}...)
	}
	if c.P75 {
		percentiles = append(percentiles, 75)
	}
	if c.P90 {
		percentiles = append(percentiles, 90)
	}
	if c.P95 {
		percentiles = append(percentiles, 95)
	}
	return uniqueFloats(percentiles)
}

func (c *Config) AddFlags(flags *pflag.FlagSet) {
	flags.BoolVarP(&c.Median, "median", "m", c.Median, "calculate median")

	flags.Float64SliceVarP(&c.Percentiles, "percentile", "p", c.Percentiles,
		"calculate percentile P (must be in the range 0..100)")
	flags.BoolVarP(&c.StdPercentiles, "std-percentiles", "s", c.StdPercentiles,
		"print standard percentiles (50, 75, 90, 95, 99)")
	flags.BoolVar(&c.P75, "p75", c.P75,
		fmt.Sprintf("calculate P%[1]d (shorthand for --p%[1]d", 75))
	flags.BoolVar(&c.P90, "p90", c.P90,
		fmt.Sprintf("calculate P%[1]d (shorthand for --p%[1]d", 90))
	flags.BoolVar(&c.P95, "p95", c.P95,
		fmt.Sprintf("calculate P%[1]d (shorthand for --p%[1]d", 95))
	flags.BoolVarP(&c.Human, "human", "u", c.Human,
		"translate human readable sizes print sizes (e.g., 1K 234M 2G)")
	flags.BoolVarP(&c.Duration, "duration", "d", c.Duration,
		"translate values as time durations (e.g., 1m11.22s, 31µs)")

	flags.BoolVar(&c.JSON, "json", c.JSON, "JSON output")
	flags.IntVar(&c.JSONIndent, "json-indent", c.JSONIndent, "indent JSON output")
}

type Result struct {
	Count int64
	Sum   *big.Float
	All   []*big.Float
}

func (c *Config) Process(rd io.Reader) (*Result, error) {
	var all []*big.Float
	var count int64
	sum := new(big.Float).SetPrec(Precision)
	human := c.Human
	duration := c.Duration
	parseErrors := 0
	needAll := c.Median || len(c.GetPercentiles()) != 0

	r := bufio.NewReaderSize(rd, 96*1024)
	var err error
	for {
		s, e := r.ReadString('\n')
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			f, perr := ParseValue(s, human, duration)
			if perr != nil {
				fmt.Fprintln(os.Stderr, "Error:", perr)
				parseErrors++
				if parseErrors >= 16 {
					err = fmt.Errorf("too many parse errors: %d", parseErrors)
					break
				}
			} else {
				sum.Add(sum, f)
				count++
				if needAll {
					all = append(all, f)
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
	if err != nil {
		return nil, err
	}
	return &Result{Count: count, Sum: sum, All: all}, nil
}

func openFile(name string) (*os.File, func() error, error) {
	if name == "-" {
		return os.Stdin, func() error { return nil }, nil
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, nil, err
	}
	return f, f.Close, nil
}

func generateShellCompletion(cmd *cobra.Command, shell string) error {
	switch shell {
	case "bash":
		return cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		return cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		return cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		return fmt.Errorf("unsupported shell: %q", shell)
	}
}

func realMain() error {
	cmd := cobra.Command{
		Use:   "gocalc: [flags] [file]...",
		Short: "gocalc calculates simple stats for numbers read from files or STDIN",
		Example: `
# Calculate sum, average, and median from STDIN:
$ printf '%s\n' 1 2 3 4 5 | ./gocalc --median

# Generate shell completion:
$ gocalc --completion [bash|zsh|fish|powershell]`[1:],

		Args: cobra.ArbitraryArgs,
	}

	var conf Config
	if term.IsTerminal(int(os.Stdout.Fd())) {
		conf.JSONIndent = 4
	}

	flags := cmd.Flags()
	flags.SortFlags = false
	conf.AddFlags(flags)

	// TODO: completion is broken
	genCompletion := flags.String("completion", "",
		"generate completion script [bash|zsh|fish|powershell]")
	cmd.RegisterFlagCompletionFunc(
		"completion",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{"bash", "zsh", "fish", "powershell"}, cobra.ShellCompDirectiveDefault
		},
	)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if *genCompletion != "" {
			return generateShellCompletion(cmd, *genCompletion)
		}

		var all []*big.Float
		var count int64
		sum := new(big.Float).SetPrec(Precision)
		if len(args) == 0 {
			args = append(args, "-")
		}
		for _, name := range args {
			f, close, err := openFile(name)
			if err != nil {
				return err
			}
			r, err := conf.Process(f)
			close()
			if err != nil {
				return err
			}
			count += r.Count
			sum.Add(sum, r.Sum)
			all = append(all, r.All...)
		}
		if count == 0 {
			return errors.New("no values") // TODO: improve error message
		}

		stats := newStats(int64(count), sum, all, conf.GetPercentiles())
		// TODO: This is kind of ugly
		stats.PrintDuration = conf.Duration

		if conf.JSON {
			enc := json.NewEncoder(os.Stdout)
			if conf.JSONIndent > 0 {
				if conf.JSONIndent == 8 {
					enc.SetIndent("", "\t")
				} else {
					enc.SetIndent("", strings.Repeat(" ", conf.JSONIndent))
				}
			}
			return enc.Encode(stats)
		}

		if _, err := stats.WriteTo(os.Stdout); err != nil {
			return err
		}
		return nil
	}

	return cmd.Execute()
}

func main() {
	if err := realMain(); err != nil {
		os.Exit(1)
	}
}
