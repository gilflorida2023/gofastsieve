package sieve

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestSimpleSieve(t *testing.T) {
	cases := []struct {
		limit uint64
		count int
		last  uint64
	}{
		{0, 0, 0},
		{1, 0, 0},
		{2, 1, 2},
		{10, 4, 7},
		{100, 25, 97},
		{1000, 168, 997},
	}
	for _, c := range cases {
		primes := SimpleSieve(c.limit)
		if len(primes) != c.count {
			t.Errorf("SimpleSieve(%d): got %d primes, want %d", c.limit, len(primes), c.count)
		}
		if c.count > 0 && primes[len(primes)-1] != c.last {
			t.Errorf("SimpleSieve(%d): last prime = %d, want %d", c.limit, primes[len(primes)-1], c.last)
		}
	}
}

func TestEratosthenesCount(t *testing.T) {
	cases := []struct {
		count int
		last  uint64
	}{
		{0, 0},
		{1, 2},
		{10, 29},
		{25, 97},
		{100, 541},
		{1000, 7919},
	}
	for _, c := range cases {
		if c.count == 0 {
			continue
		}
		limit := estimateNthPrime(uint64(c.count))
		s := NewEratosthenes(limit)
		var last uint64
		n := 0
		for p := range s.Primes() {
			last = p
			n++
			if n >= c.count {
				break
			}
		}
		if last != c.last {
			t.Errorf("Eratosthenes(%d): last prime = %d, want %d", c.count, last, c.last)
		}
	}
}

func TestStreamHasherFormat(t *testing.T) {
	hasher := NewStreamHasher()
	hasher.WriteInt(2)
	hasher.WriteInt(3)
	hasher.WriteInt(5)
	got := hasher.HexSum()

	// Expected: SHA-256 of "2\n3\n5\n"
	h := sha256.New()
	h.Write([]byte("2\n3\n5\n"))
	expected := fmt.Sprintf("%x", h.Sum(nil))

	if got != expected {
		t.Errorf("hash = %s, want %s", got, expected)
	}
}

func TestWheel2310(t *testing.T) {
	w := NewWheel(2310)
	if w.SpokeCount != 480 {
		t.Errorf("SpokeCount = %d, want 480", w.SpokeCount)
	}
	if len(w.Spokes) != 480 {
		t.Errorf("len(Spokes) = %d, want 480", len(w.Spokes))
	}
	if w.Spokes[0] != 1 {
		t.Errorf("first spoke = %d, want 1", w.Spokes[0])
	}
	// 121 = 11^2, should NOT be a spoke (divisible by wheel prime 11)
	if w.ResidueToBit[121] >= 0 {
		t.Error("121 should not be a spoke residue (divisible by 11)")
	}
	// 169 = 13^2, should be a spoke (not divisible by any wheel prime)
	if w.ResidueToBit[169] < 0 {
		t.Error("169 should be a spoke residue (coprime to 2,3,5,7,11)")
	}
}

func TestEratosthenesWheel2310Count(t *testing.T) {
	cases := []struct {
		limit uint64
		count int
	}{
		{100, 25},
		{1000, 168},
		{10000, 1229},
	}
	for _, c := range cases {
		s := NewEratosthenesWithWheel(c.limit, 2310)
		n := 0
		s.ForEachPrime(func(uint64) bool { n++; return true })
		if n != c.count {
			t.Errorf("wheel-2310 limit=%d: got %d primes, want %d", c.limit, n, c.count)
		}
	}
}

func TestHashN(t *testing.T) {
	// Verify against Math-KAT manifest checkpoint hashes
	cases := []struct {
		n      uint64
		expect string
	}{
		{10, "dc8c353498db9b9bb1161eab32f94206df30e014947ae64482851f3fafed07ff"},
		{100, "5991e67de21b5e0aac4191be06e69b5e32e8431858a108c4029906aaa96a1371"},
		{1000, "18ac898998c81cb9eb52d37be6cd452a3b19babedbdd5cc6e8ffff20e7c2b048"},
	}
	for _, c := range cases {
		got, err := HashN(c.n)
		if err != nil {
			t.Fatalf("HashN(%d): %v", c.n, err)
		}
		if got != c.expect {
			t.Errorf("HashN(%d) = %s, want %s", c.n, got, c.expect)
		}
	}
}

func BenchmarkEratosthenes100k(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s := NewEratosthenes(1_300_000) // enough for ~100k primes
		for range s.Primes() {
		}
	}
}

func BenchmarkForEachPrime100k(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s := NewEratosthenes(1_300_000)
		s.ForEachPrime(func(uint64) bool { return true })
	}
}

func BenchmarkHashN100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HashN(100)
	}
}

func BenchmarkHashN1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HashN(1000)
	}
}

func BenchmarkHashN10000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		HashN(10000)
	}
}
