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

func spokeResidueTest(t *testing.T, w *Wheel, mod uint64, residue uint64, shouldBeSpoke bool) {
	t.Helper()
	label := fmt.Sprintf("wheel-%d residue %d", mod, residue)
	if shouldBeSpoke && w.ResidueToBit[residue] < 0 {
		t.Errorf("%s should be a spoke (coprime to wheel primes)", label)
	}
	if !shouldBeSpoke && w.ResidueToBit[residue] >= 0 {
		t.Errorf("%s should NOT be a spoke (divisible by wheel prime)", label)
	}
}

func TestWheel30(t *testing.T) {
	w := NewWheel(30)
	if w.SpokeCount != 8 {
		t.Errorf("SpokeCount = %d, want 8", w.SpokeCount)
	}
	if len(w.Spokes) != 8 {
		t.Errorf("len(Spokes) = %d, want 8", len(w.Spokes))
	}
	if w.Spokes[0] != 1 {
		t.Errorf("first spoke = %d, want 1", w.Spokes[0])
	}
	// Wheel-30 has primes {2,3,5}, residues in [0,29]
	// 25 = 5^2, should NOT be a spoke (divisible by 5)
	spokeResidueTest(t, w, 30, 25, false)
	// 9 = 3^2, should NOT be a spoke (divisible by 3)
	spokeResidueTest(t, w, 30, 9, false)
	// 7 is coprime to 2,3,5 → should be a spoke
	spokeResidueTest(t, w, 30, 7, true)
	// 1 is always the first spoke
	spokeResidueTest(t, w, 30, 1, true)
	// 0 is never a spoke (even)
	spokeResidueTest(t, w, 30, 0, false)
	// 17 is coprime to 2,3,5 → should be a spoke
	spokeResidueTest(t, w, 30, 17, true)
	// Expected spokes for wheel-30: {1, 7, 11, 13, 17, 19, 23, 29}
	expected := []uint64{1, 7, 11, 13, 17, 19, 23, 29}
	if len(w.Spokes) == len(expected) {
		for i := range w.Spokes {
			if w.Spokes[i] != expected[i] {
				t.Errorf("Spokes[%d] = %d, want %d", i, w.Spokes[i], expected[i])
			}
		}
	}
}

func TestWheel210(t *testing.T) {
	w := NewWheel(210)
	if w.SpokeCount != 48 {
		t.Errorf("SpokeCount = %d, want 48", w.SpokeCount)
	}
	if len(w.Spokes) != 48 {
		t.Errorf("len(Spokes) = %d, want 48", len(w.Spokes))
	}
	if w.Spokes[0] != 1 {
		t.Errorf("first spoke = %d, want 1", w.Spokes[0])
	}
	// Wheel-210 has primes {2,3,5,7}, so residues divisible by 2,3,5,7 are not spokes
	// 121 = 11^2, should BE a spoke (coprime to 2,3,5,7)
	spokeResidueTest(t, w, 210, 121, true)
	// 169 = 13^2, should be a spoke (coprime to 2,3,5,7)
	spokeResidueTest(t, w, 210, 169, true)
	// 143 = 11*13, should be a spoke (coprime to 2,3,5,7)
	spokeResidueTest(t, w, 210, 143, true)
	// 49 = 7^2, should NOT be a spoke (divisible by 7)
	spokeResidueTest(t, w, 210, 49, false)
	// 121 is explicitly listed in the DefaultWheel210 Spokes slice — verify it's there
	found := false
	for _, s := range w.Spokes {
		if s == 121 {
			found = true
			break
		}
	}
	if !found {
		t.Error("wheel-210 spokes should include 121 (coprime to 2,3,5,7)")
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

func TestCrossWheelConsistency(t *testing.T) {
	mods := []uint64{2, 6, 30, 210, 2310}
	limits := []uint64{100, 1000, 10000}
	for _, limit := range limits {
		var counts []int
		for _, mod := range mods {
			s := NewEratosthenesWithWheel(limit, mod)
			n := 0
			s.ForEachPrime(func(uint64) bool { n++; return true })
			counts = append(counts, n)
		}
		// All wheels must produce the same prime count
		for i := 1; i < len(counts); i++ {
			if counts[i] != counts[0] {
				t.Errorf("limit=%d: wheel-%d gave %d primes, wheel-%d gave %d",
					limit, mods[0], counts[0], mods[i], counts[i])
			}
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

func TestEratosthenesSmallLimits(t *testing.T) {
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
		s := NewEratosthenes(c.limit)
		n := 0
		s.ForEachPrime(func(uint64) bool { n++; return true })
		if n != c.count {
			t.Errorf("limit=%d: got %d primes, want %d", c.limit, n, c.count)
		}
		// Also test channel-based Primes()
		s2 := NewEratosthenes(c.limit)
		n2 := 0
		for range s2.Primes() {
			n2++
		}
		if n2 != c.count {
			t.Errorf("Primes() limit=%d: got %d primes, want %d", c.limit, n2, c.count)
		}
	}
}

func TestForEachPrimeEarlyReturn(t *testing.T) {
	s := NewEratosthenes(100000)
	count := 0
	s.ForEachPrime(func(p uint64) bool {
		count++
		return count < 50 // stop after 50 primes
	})
	if count != 50 {
		t.Errorf("early return: got %d primes, want 50", count)
	}
}

func BenchmarkEratosthenes100k(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s := NewEratosthenes(1_300_000)
		for range s.Primes() {
		}
	}
}

func BenchmarkWheelComparison(b *testing.B) {
	limit := uint64(10_000_000)
	mods := []uint64{2, 6, 30, 210, 2310}
	for _, mod := range mods {
		b.Run(fmt.Sprintf("wheel-%d", mod), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s := NewEratosthenesWithWheel(limit, mod)
				s.ForEachPrime(func(uint64) bool { return true })
			}
		})
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
