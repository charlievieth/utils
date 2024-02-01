package main

import (
	crand "crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func parsePositionalArgs(_ *cobra.Command, args []string) (count, first, last *int64, _ error) {
	if len(args) == 0 {
		return nil, nil, nil, errors.New(`missing required argument "count"`)
	}
	if len(args) > 3 {
		return nil, nil, nil, errors.New("too many arguments")
	}
	type value struct {
		n    **int64
		name string
	}
	for i, v := range []value{{&count, "count"}, {&first, "first"}, {&last, "last"}} {
		if i >= len(args) {
			break
		}
		n, err := strconv.ParseInt(args[i], 10, 64)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("invalid argument %q: %w", v.name, err)
		}
		if n < 0 {
			return nil, nil, nil, fmt.Errorf("non-positive argument %q: %d", v.name, n)
		}
		*v.n = &n
	}
	if count == nil {
		return nil, nil, nil, errors.New(`missing required argument "count"`)
	}
	if first != nil && last != nil && *first >= *last {
		return nil, nil, nil, fmt.Errorf(`argument "first" greater than "last": %d >= %d`,
			*first, *last)
	}
	return count, first, last, nil
}

func validArgs(_ *cobra.Command, args []string) error {
	_, _, _, err := parsePositionalArgs(nil, args)
	return err
}

func newRand() (*rand.Rand, error) {
	randSeed, err := crand.Int(crand.Reader, big.NewInt(1<<63-1))
	if err != nil {
		return nil, err
	}
	return rand.New(rand.NewSource(randSeed.Int64())), nil
}

func randomPrimes(count, min, max int64) ([]int64, error) {
	if min < 0 {
		return nil, fmt.Errorf("non-positive minimum: %d", min)
	}
	if max < 0 {
		return nil, fmt.Errorf("non-positive maximum: %d", max)
	}
	if min >= max {
		return nil, fmt.Errorf("minimum %d must be less than maximum: %d", min, max)
	}
	if count > max-min {
		return nil, fmt.Errorf("impossible to generate %d primes in range %d..%d",
			count, min, max)
	}

	// Make sure we can generate count primes
	// TODO: there are definitely faster and better ways to do this
	if (max-min)/count < 1_000 {
		bn := big.NewInt(0)
		c := count
		for n := min; n < max && c > 0; n++ {
			if n&1 == 0 && n != 2 {
				continue
			}
			if bn.SetInt64(n).ProbablyPrime(1) {
				c--
			}
		}
		if c > 0 {
			return nil, fmt.Errorf("impossible to generate %d primes in range %d..%d",
				count, min, max)
		}
	}

	rr, err := newRand()
	if err != nil {
		return nil, err
	}

	seen := make(map[int64]struct{}, count)

	bn := big.NewInt(0)
	primes := make([]int64, 0, count)
	for int64(len(primes)) < count {
		n := rr.Int63n(max-min) + min
		if n&1 == 0 && n != 2 {
			n++
		}
		if bn.SetInt64(n).ProbablyPrime(1) {
			if _, ok := seen[n]; !ok {
				seen[n] = struct{}{}
				primes = append(primes, n)
			}
		}
		if int64(len(seen)) >= (max-min)/2 && max-min > 1 {
			return primes, fmt.Errorf("impossible to generate %d primes in range %d..%d",
				count, min, max)
		}
	}
	return primes, nil
}

func sequentialPrimes(count, first, last int64) ([]int64, error) {
	// TODO: support negative count (return all in range)
	primes := make([]int64, 0, count)
	for i := first; i <= last && int64(len(primes)) < count; i++ {
		if big.NewInt(int64(i)).ProbablyPrime(1) {
			primes = append(primes, i)
		}
	}
	if int64(len(primes)) < count {
		return primes, fmt.Errorf("impossible to generate %d primes in range %d..%d",
			count, first, last)
	}
	return primes, nil
}

func main() {
	cmd := cobra.Command{
		Use:   "genprimes [options]... count [first] [last|max]",
		Short: "Generate count prime numbers in range first..last",
		Args:  cobra.MatchAll(cobra.MinimumNArgs(1), cobra.MaximumNArgs(3), validArgs),
	}
	ff := cmd.Flags()
	random := ff.BoolP("rand", "r", false, "generate random primes")
	join := ff.BoolP("join", "j", false, "print joined results")
	suffix := ff.StringP("suffix", "s", "", "suffix to append to results")
	_ = suffix
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		var count, first, last int64
		for i, p := range []*int64{&count, &first, &last} {
			if i >= len(args) {
				break
			}
			n, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				return fmt.Errorf("parsing arg %d: %w", i+1, err)
			}
			*p = n
		}
		if count > 0 && last == 0 {
			last = 1<<63 - 1
		}
		var primes []int64
		if *random {
			primes, err = randomPrimes(count, first, last)
		} else {
			primes, err = sequentialPrimes(count, first, last)
		}
		// TODO: print primes on error
		if err != nil {
			return err
		}
		a := make([]string, len(primes))
		for i, n := range primes {
			a[i] = strconv.FormatInt(n, 10)
		}
		if *suffix != "" {
			s := *suffix
			for i := range a {
				a[i] += s
			}
		}
		if *join {
			if _, err := fmt.Println(strings.Join(a, " ")); err != nil {
				return err
			}
		} else {
			for _, s := range a {
				if _, err := fmt.Println(s); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
