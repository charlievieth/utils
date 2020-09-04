package main

import "testing"

func benchmarkWalk(b *testing.B, size int) {
	ll := NewList()
	for i := 0; i < size; i++ {
		ll.PushFront(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ll.Walk()
	}
}

func benchmarkWalkFast(b *testing.B, size int) {
	ll := NewList()
	for i := 0; i < size; i++ {
		ll.PushFront(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ll.WalkFast()
	}
}

func BenchmarkWalkFast_1k(b *testing.B) {
	benchmarkWalkFast(b, 1000)
}

func BenchmarkWalkFast_10k(b *testing.B) {
	benchmarkWalkFast(b, 10_000)
}

func BenchmarkWalkFast_100k(b *testing.B) {
	benchmarkWalkFast(b, 100_000)
}

func BenchmarkWalkFast_1m(b *testing.B) {
	benchmarkWalkFast(b, 1_000_000)
}

func BenchmarkWalk_1k(b *testing.B) {
	benchmarkWalk(b, 1000)
}

func BenchmarkWalk_10k(b *testing.B) {
	benchmarkWalk(b, 10_000)
}

func BenchmarkWalk_100k(b *testing.B) {
	benchmarkWalk(b, 100_000)
}

func BenchmarkWalk_1m(b *testing.B) {
	benchmarkWalk(b, 1_000_000)
}
