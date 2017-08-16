package main

import (
	"math/rand"

	"testing"
)

var IntSet []int
var CheckSet []int

func init() {
	rand.Seed(1)
	IntSet = make([]int, 4096)
	seen := make(map[int]bool, len(IntSet))
	for i := 0; i < len(IntSet); i++ {
		n := rand.Int()
		seen[n] = true
		IntSet[i] = n
	}
	CheckSet = make([]int, len(IntSet)*2)
	n := copy(CheckSet, IntSet)
	for i := n; i < len(CheckSet); i++ {
		n := rand.Int()
		for ; seen[n]; n = rand.Int() {
		}
		CheckSet[i] = n
	}
}

func BenchmarkSparseHas(b *testing.B) {
	var s Sparse
	for _, n := range IntSet {
		s.Insert(n)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, x := range CheckSet {
			_ = s.Has(x)
		}
	}
}

func BenchmarkMapHas(b *testing.B) {
	m := make(map[int]struct{})
	for _, n := range IntSet {
		m[n] = struct{}{}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, x := range CheckSet {
			_ = m[x]
		}
	}
}
