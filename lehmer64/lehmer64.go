package lehmer64

import "math/bits"

// https://lemire.me/blog/2019/03/19/the-fastest-conventional-random-number-generator-that-can-pass-big-crush/
// https://github.com/lemire/testingRNG/blob/master/source/lehmer64.h
type Lehmer64 struct {
	hi, lo uint64
}

func splitMix64(seed uint64) uint64 {
	z := (seed + 0x9E3779B97F4A7C15) // golden gamma
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

func splitMix64Stateless(seed, offset uint64) uint64 {
	seed += offset * 0x9E3779B97F4A7C15 // golden gamma
	return splitMix64(seed)
}

func NewLehmer64(seed uint64) *Lehmer64 {
	return &Lehmer64{
		hi: splitMix64Stateless(seed, 0),
		lo: splitMix64Stateless(seed, 1),
	}
}

func (l *Lehmer64) Uint64() uint64 {
	const c = 0xda942042e4dd58b5
	hl := l.hi * c
	l.hi, l.lo = bits.Mul64(l.lo, c)
	l.hi += hl
	return l.hi
}

func (l *Lehmer64) Int63() int64 {
	return int64(l.Uint64() & ((1 << 63) - 1))
}

func (l *Lehmer64) Seed(seed int64) {
	l.hi = splitMix64Stateless(uint64(seed), 0)
	l.lo = splitMix64Stateless(uint64(seed), 1)
}
