package sieve

import "math"

// SimpleSieve returns all primes up to limit using a basic sieve.
// Used for generating base primes up to sqrt(N).
func SimpleSieve(limit uint64) []uint64 {
	if limit < 2 {
		return nil
	}
	n := int(limit + 1)
	sqrt := int(math.Sqrt(float64(limit)))
	comp := make([]bool, n)
	for i := 2; i <= sqrt; i++ {
		if !comp[i] {
			for j := i * i; j < n; j += i {
				comp[j] = true
			}
		}
	}
	var primes []uint64
	for i := 2; i < n; i++ {
		if !comp[i] {
			primes = append(primes, uint64(i))
		}
	}
	return primes
}
