package sieve

import (
	"math"
	"math/bits"
	"runtime"
)

// preSieveCutoff is the largest prime handled by pre-sieve masks.
// Primes > lastWheelPrime and <= preSieveCutoff are masked via pre-computed
// bit patterns at each block, avoiding the inner marking loop for dense multiples.
const preSieveCutoff = 100

// segSpanMul scales the segment span to reduce per-segment overhead.
// Higher values = fewer, larger segments = fewer goroutine creations
// and buffer allocations. Must be ≥ 1.
// Wheel-30030 uses smaller multiplier to limit buffer size.
func segSpanMulForWheel(spokeCount int) int {
	if spokeCount >= 5000 {
		return 32 // wheel-30030: smaller segments to fit in cache
	}
	return 256 // wheel-2310 and smaller
}

type preSieveMask struct {
	prime uint64
	masks []uint64 // p masks, each of wordStride words
}

// BitPackedEratosthenes implements a wheel-based segmented prime sieve
// using a bit-packed segment buffer.
type BitPackedEratosthenes struct {
	limit         uint64
	segSpan       uint64
	wheel         *Wheel
	wordStride    uint64 // uint64 words per wheel block = ceil(SpokeCount / 64)
	preSieveMasks []preSieveMask
}

// NewBitPackedEratosthenes creates a bit-packed segmented sieve.
func NewBitPackedEratosthenes(limit, wheelMod uint64) *BitPackedEratosthenes {
	w := NewWheel(wheelMod)
	ws := uint64((w.SpokeCount + 63) / 64)
	mul := segSpanMulForWheel(w.SpokeCount)
	e := &BitPackedEratosthenes{
		limit:      limit,
		segSpan:    ((262144 * uint64(mul)) / uint64(w.SpokeCount)) * w.Modulus,
		wheel:      w,
		wordStride: ws,
	}
	e.initPreSieve()
	return e
}

func (e *BitPackedEratosthenes) initPreSieve() {
	w := e.wheel
	ws := e.wordStride
	lastWheelPrime := w.WheelPrimes[len(w.WheelPrimes)-1]

	allPrimes := SimpleSieve(preSieveCutoff)
	for _, p := range allPrimes {
		if p <= lastWheelPrime {
			continue
		}
		masks := make([]uint64, p*ws)
		for b := uint64(0); b < p; b++ {
			for si, s := range w.Spokes {
				if (b*w.Modulus+s)%p == 0 {
					wi := uint64(si) / 64
					bi := uint64(si) % 64
					masks[b*ws+wi] |= 1 << bi
				}
			}
		}
		e.preSieveMasks = append(e.preSieveMasks, preSieveMask{prime: p, masks: masks})
	}
}

// ForEachPrime calls fn for each prime up to the limit.
// Returns early if fn returns false.
// Automatically parallelizes for non-trivial limits on multi-core systems.
func (e *BitPackedEratosthenes) ForEachPrime(fn func(uint64) bool) {
	if e.limit >= minParallelLimit && runtime.GOMAXPROCS(0) > 1 {
		e.parallelGenerate(fn)
	} else {
		e.generate(fn)
	}
}

// ForEachPrimeSequential calls fn for each prime using the sequential algorithm.
func (e *BitPackedEratosthenes) ForEachPrimeSequential(fn func(uint64) bool) {
	e.generate(fn)
}

func (e *BitPackedEratosthenes) generate(emit func(uint64) bool) {
	w := e.wheel
	ws := e.wordStride

	for _, p := range w.WheelPrimes {
		if p > e.limit {
			return
		}
		if !emit(p) {
			return
		}
	}

	lastWheelPrime := w.WheelPrimes[len(w.WheelPrimes)-1]
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
		wheels = append(wheels, NewWheelPrime(p, w))
	}

	lo := sqrtLimit + 1

	var buf []uint64
	for lo <= e.limit {
		hi := lo + e.segSpan - 1
		if hi > e.limit || hi < lo {
			hi = e.limit
		}

		firstBlock := lo / w.Modulus
		lastBlock := hi / w.Modulus
		numBlocks := lastBlock - firstBlock + 1
		segLen := numBlocks * ws

		if cap(buf) < int(segLen) {
			buf = make([]uint64, segLen)
		}
		buf = buf[:segLen]
		// Initialize: all bits set = every candidate alive
		for i := range buf {
			buf[i] = ^uint64(0)
		}
		// Mask trailing bits in the last word beyond SpokeCount
		if lastBits := uint(w.SpokeCount % 64); lastBits > 0 && segLen > 0 {
			buf[segLen-1] &= (1 << lastBits) - 1
		}

		// Phase 0: pre-sieve small primes via pre-computed masks
		// Only primes ≤ sqrtLimit are pre-sieved. Primes > sqrtLimit are
		// survivors in the sieve range; marking them as composite is incorrect.
		for _, psm := range e.preSieveMasks {
			p := psm.prime
			if p > sqrtLimit {
				continue
			}
			masks := psm.masks
			for bi := uint64(0); bi < numBlocks; bi++ {
				bMod := (firstBlock + bi) % p
				off := bi * ws
				moff := bMod * ws
				for wi := uint64(0); wi < ws; wi++ {
					buf[off+wi] &^= masks[moff+wi]
				}
			}
		}

		// Phase 1: mark multiples of wheel primes (small primes handled by Phase 0)
		for _, wp := range wheels {
			p := wp.Prime
			if p <= preSieveCutoff {
				continue
			}
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

			block := m / w.Modulus
			r := m % w.Modulus

			for m <= hi {
				if ri := w.ResidueToBit[r]; ri >= 0 {
					wordIdx := uint64(ri) / 64
					bitIdx := uint64(ri) % 64
					buf[(block-firstBlock)*ws+wordIdx] &^= 1 << bitIdx
				}
				m += p
				block += wp.BlkStep
				r += wp.Step
				if r >= w.Modulus {
					r -= w.Modulus
					block++
				}
			}
		}

		// Phase 2: scan survivors and emit
		for bi := uint64(0); bi < numBlocks; bi++ {
			base := bi * ws
			blockBase := (firstBlock + bi) * w.Modulus
			for wi := uint64(0); wi < ws; wi++ {
				word := buf[base+wi]
				if word == 0 {
					continue
				}
			bitBase := wi * 64
			for word != 0 {
				bit := uint64(bits.TrailingZeros64(word))
				word &^= 1 << bit
				si := int(bitBase + bit)
				if si >= w.SpokeCount {
					break
				}
				n := blockBase + w.Spokes[si]
				if n < lo || n > hi {
					continue
				}
				if !emit(n) {
					return
				}
			}
			}
		}

		lo = hi + 1
	}
}
