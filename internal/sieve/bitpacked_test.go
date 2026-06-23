package sieve

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

// crossValidate verifies that bit-packed and byte-based sieves produce
// the same primes at the given limit for all wheel moduli.
func crossValidate(t *testing.T, limit uint64) {
	mods := []uint64{2, 6, 30, 210, 2310, 30030}
	for _, mod := range mods {
		// Reference: byte-based sieve
		ref := NewEratosthenesWithWheel(limit, mod)
		var refPrimes []uint64
		ref.ForEachPrime(func(p uint64) bool {
			refPrimes = append(refPrimes, p)
			return true
		})

		// Bit-packed sieve
		bp := NewBitPackedEratosthenes(limit, mod)
		var bpPrimes []uint64
		bp.ForEachPrime(func(p uint64) bool {
			bpPrimes = append(bpPrimes, p)
			return true
		})

		if len(refPrimes) != len(bpPrimes) {
			t.Errorf("wheel-%d limit=%d: byte gave %d primes, bit gave %d primes",
				mod, limit, len(refPrimes), len(bpPrimes))
			continue
		}
		for i := range refPrimes {
			if refPrimes[i] != bpPrimes[i] {
				t.Errorf("wheel-%d limit=%d: byte[%d]=%d != bit[%d]=%d",
					mod, limit, i, refPrimes[i], i, bpPrimes[i])
				break
			}
		}
	}
}

func TestBitPackedCrossValidation(t *testing.T) {
	limits := []uint64{100, 1000, 10000, 100000}
	for _, limit := range limits {
		crossValidate(t, limit)
	}
}

func TestBitPackedSmallLimits(t *testing.T) {
	cases := []struct {
		limit uint64
		count int
	}{
		{0, 0},
		{1, 0},
		{2, 1},
		{3, 2},
		{10, 4},
	}
	for _, c := range cases {
		s := NewBitPackedEratosthenes(c.limit, 210)
		n := 0
		s.ForEachPrime(func(uint64) bool { n++; return true })
		if n != c.count {
			t.Errorf("limit=%d: got %d primes, want %d", c.limit, n, c.count)
		}
	}
}

func TestBitPackedHashConsistency(t *testing.T) {
	// Bit-packed sieve must produce identical hashes to byte-based sieve
	// at 10^3 and 10^6 for all wheel moduli.
	type testCase struct {
		limit uint64
		mod   uint64
	}
	cases := []testCase{
		{1000, 2}, {1000, 6}, {1000, 30}, {1000, 210}, {1000, 2310}, {1000, 30030},
		{1000000, 2}, {1000000, 6}, {1000000, 30}, {1000000, 210}, {1000000, 2310}, {1000000, 30030},
		{1000000000, 2}, {1000000000, 6}, {1000000000, 30}, {1000000000, 210}, {1000000000, 2310}, {1000000000, 30030},
	}
	for _, c := range cases {
		// Generate reference hash with byte-based sieve
		h := sha256.New()
		sRef := NewEratosthenesWithWheel(c.limit, c.mod)
		sRef.ForEachPrime(func(p uint64) bool {
			fmt.Fprintf(h, "%d\n", p)
			return true
		})
		refHash := fmt.Sprintf("%x", h.Sum(nil))

		// Generate hash with bit-packed sieve
		h2 := sha256.New()
		sBp := NewBitPackedEratosthenes(c.limit, c.mod)
		sBp.ForEachPrime(func(p uint64) bool {
			fmt.Fprintf(h2, "%d\n", p)
			return true
		})
		bpHash := fmt.Sprintf("%x", h2.Sum(nil))

		if bpHash != refHash {
			t.Errorf("wheel-%d limit=%d: byte hash %s, bit hash %s (MISMATCH!)",
				c.mod, c.limit, refHash, bpHash)
		}
	}
}

func TestBitPackedForEachPrimeEarlyReturn(t *testing.T) {
	s := NewBitPackedEratosthenes(100000, 210)
	count := 0
	s.ForEachPrime(func(p uint64) bool {
		count++
		return count < 50
	})
	if count != 50 {
		t.Errorf("early return: got %d primes, want 50", count)
	}
}

func BenchmarkBitPackedWheel210(b *testing.B) {
	limit := uint64(10_000_000)
	for i := 0; i < b.N; i++ {
		s := NewBitPackedEratosthenes(limit, 210)
		s.ForEachPrime(func(uint64) bool { return true })
	}
}

func BenchmarkBitPackedWheel2310(b *testing.B) {
	limit := uint64(10_000_000)
	for i := 0; i < b.N; i++ {
		s := NewBitPackedEratosthenes(limit, 2310)
		s.ForEachPrime(func(uint64) bool { return true })
	}
}

func BenchmarkBitPackedVsByteComparison(b *testing.B) {
	limit := uint64(10_000_000)
	mods := []uint64{2, 6, 30, 210, 2310, 30030}
	for _, mod := range mods {
		b.Run(fmt.Sprintf("byte-wheel-%d", mod), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s := NewEratosthenesWithWheel(limit, mod)
				s.ForEachPrime(func(uint64) bool { return true })
			}
		})
		b.Run(fmt.Sprintf("bit-wheel-%d", mod), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s := NewBitPackedEratosthenes(limit, mod)
				s.ForEachPrime(func(uint64) bool { return true })
			}
		})
	}
}

func BenchmarkParallelComparison(b *testing.B) {
	for _, limit := range []uint64{10_000_000, 50_000_000, 100_000_000, 200_000_000} {
		b.Run(fmt.Sprintf("par-wheel-2310-%d", limit), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s := NewBitPackedEratosthenes(limit, 2310)
				s.ForEachPrime(func(uint64) bool { return true })
			}
		})
		b.Run(fmt.Sprintf("seq-wheel-2310-%d", limit), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s := NewBitPackedEratosthenes(limit, 2310)
				s.ForEachPrimeSequential(func(uint64) bool { return true })
			}
		})
		b.Run(fmt.Sprintf("par-wheel-30030-%d", limit), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s := NewBitPackedEratosthenes(limit, 30030)
				s.ForEachPrime(func(uint64) bool { return true })
			}
		})
		b.Run(fmt.Sprintf("seq-wheel-30030-%d", limit), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s := NewBitPackedEratosthenes(limit, 30030)
				s.ForEachPrimeSequential(func(uint64) bool { return true })
			}
		})
	}
}
