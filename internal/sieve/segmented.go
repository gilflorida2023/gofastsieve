package sieve

import "math"

// Eratosthenes implements a wheel-based segmented prime sieve.
type Eratosthenes struct {
	limit   uint64
	segSpan uint64
	wheel   *Wheel
}

// NewEratosthenes creates a segmented sieve for primes up to limit using wheel-210.
func NewEratosthenes(limit uint64) *Eratosthenes {
	return NewEratosthenesWithWheel(limit, 210)
}

// NewEratosthenesWithWheel creates a segmented sieve with the given wheel modulus.
func NewEratosthenesWithWheel(limit, wheelMod uint64) *Eratosthenes {
	w := NewWheel(wheelMod)
	return &Eratosthenes{
		limit:   limit,
		segSpan: (262144 / uint64(w.SpokeCount)) * w.Modulus,
		wheel:   w,
	}
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
	// Emit wheel-defining primes (2,3,5,7 for 210; 2,3,5 for 30; etc.)
	for _, p := range e.wheel.WheelPrimes {
		if p > e.limit {
			return
		}
		if !emit(p) {
			return
		}
	}

	lastWheelPrime := e.wheel.WheelPrimes[len(e.wheel.WheelPrimes)-1]
	if e.limit <= lastWheelPrime {
		return
	}

	sqrtLimit := uint64(math.Sqrt(float64(e.limit)))
	basePrimes := SimpleSieve(sqrtLimit)

	var wheels []WheelPrime
	for _, p := range basePrimes {
		if p <= lastWheelPrime {
			continue
		}
		if !emit(p) {
			return
		}
		wheels = append(wheels, NewWheelPrime(p, e.wheel))
	}

	stride := uint64(e.wheel.SpokeCount)
	lo := sqrtLimit + 1

	var buf []byte
	for lo <= e.limit {
		hi := lo + e.segSpan - 1
		if hi > e.limit || hi < lo {
			hi = e.limit
		}

		firstBlock := lo / e.wheel.Modulus
		lastBlock := hi / e.wheel.Modulus
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

			block := m / e.wheel.Modulus
			r := m % e.wheel.Modulus

			for m <= hi {
				if ri := e.wheel.ResidueToBit[r]; ri >= 0 {
					buf[(block-firstBlock)*stride+uint64(ri)] = 0
				}
				m += p
				block += wp.BlkStep
				r += wp.Step
				if r >= e.wheel.Modulus {
					r -= e.wheel.Modulus
					block++
				}
			}
		}

		// Phase 2: scan survivors and emit
		for bi := uint64(0); bi < numBlocks; bi++ {
			base := bi * stride
			for si := 0; si < e.wheel.SpokeCount; si++ {
				if buf[base+uint64(si)] == 0 {
					continue
				}
				n := (firstBlock+bi)*e.wheel.Modulus + e.wheel.Spokes[si]
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
