package main

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"os"
)

func SessionID() (uint64, error) {
	var max big.Int
	max.SetInt64(math.MaxInt64)
	n, err := rand.Int(rand.Reader, &max)
	if err != nil {
		return 0, err
	}
	return n.Uint64(), nil
}

func main() {
	n, err := SessionID()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	fmt.Println(n)
}
