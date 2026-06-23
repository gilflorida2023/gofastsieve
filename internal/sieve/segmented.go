package sieve

import "math"

// Eratosthenes implements a wheel-210 segmented prime sieve.
type Eratosthenes struct {
	limit   uint64
	segSpan uint64 // number of integers per segment
}

// NewEratosthenes creates a segmented sieve for primes up to limit.
func NewEratosthenes(limit uint64) *Eratosthenes {
	return &Eratosthenes{
		limit:   limit,
		segSpan: defaultSegSpan(limit),
	}
}

func defaultSegSpan(uint64) uint64 {
	return (262144 / WheelSpoke) * WheelMod
}

// ForEachPrime calls fn for each prime up to the limit.
// Returns early if fn returns false.
func (e *Eratosthenes) ForEachPrime(fn func(uint64) bool) {
	e.generate(fn)
}

// Primes returns a channel of all primes up to the limit.
func (e *Eratosthenes) Primes() <-chan uint64 {
	out := make(chan uint64, 256)
	go func() {
		e.generate(func(n uint64) bool { out <- n; return true })
		close(out)
	}()
	return out
}

func (e *Eratosthenes) generate(emit func(uint64) bool) {
	if e.limit < 2 {
		return
	}
	if !emit(2) {
		return
	}
	if e.limit < 3 {
		return
	}
	if !emit(3) {
		return
	}
	if e.limit < 5 {
		return
	}
	if !emit(5) {
		return
	}
	if e.limit < 7 {
		return
	}
	if !emit(7) {
		return
	}
	if e.limit < 11 {
		return
	}

	sqrtLimit := uint64(math.Sqrt(float64(e.limit)))
	basePrimes := SimpleSieve(sqrtLimit)

	var wheels []WheelPrime
	for _, p := range basePrimes {
		if p <= 7 {
			continue
		}
		if !emit(p) {
			return
		}
		wheels = append(wheels, NewWheelPrime(p))
	}

	stride := uint64(WheelSpoke)
	lo := sqrtLimit + 1

	var buf []byte
	for lo <= e.limit {
		hi := lo + e.segSpan - 1
		if hi > e.limit || hi < lo {
			hi = e.limit
		}

		firstBlock := lo / WheelMod
		lastBlock := hi / WheelMod
		numBlocks := lastBlock - firstBlock + 1
		segLen := numBlocks * stride

		if cap(buf) < int(segLen) {
			buf = make([]byte, segLen)
		}
		buf = buf[:segLen]
		for i := range buf {
			buf[i] = 0xFF
		}

		// Phase 1: mark multiples of wheel primes
		for _, wp := range wheels {
			p := wp.Prime
			if p > hi {
				continue
			}

			m := lo
			if rem := m % p; rem != 0 {
				m += p - rem
			}
			if m < p*p {
				m = p * p
			}
			if m > hi {
				continue
			}

			block := m / WheelMod
			r := m % WheelMod

			for m <= hi {
				if ri := ResidueToBit[r]; ri >= 0 {
					buf[(block-firstBlock)*stride+uint64(ri)] = 0
				}
				m += p
				block += wp.BlkStep
				r += wp.Step
				if r >= WheelMod {
					r -= WheelMod
					block++
				}
			}
		}

		// Phase 2: scan survivors and emit
		for bi := uint64(0); bi < numBlocks; bi++ {
			base := bi * stride
			for si := 0; si < WheelSpoke; si++ {
				if buf[base+uint64(si)] == 0 {
					continue
				}
				n := (firstBlock+bi)*WheelMod + Spokes[si]
				if n < lo || n > hi {
					continue
				}
				if !emit(n) {
					return
				}
			}
		}

		lo = hi + 1
	}
}
