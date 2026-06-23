package sieve

import (
	"math"
	"math/bits"
)

// BitPackedEratosthenes implements a wheel-based segmented prime sieve
// using a bit-packed segment buffer. Instead of one byte per spoke,
// it packs all spokes of a block into uint64 words, using bits.TrailingZeros64
// to efficiently scan only surviving residues.
//
// This provides the greatest benefit for large wheels (especially wheel-2310
// with 480 spokes packed into 8 uint64 words) where the Phase 2 scan loop
// is the primary bottleneck.
type BitPackedEratosthenes struct {
	limit      uint64
	segSpan    uint64
	wheel      *Wheel
	wordStride uint64 // uint64 words per wheel block = ceil(SpokeCount / 64)
}

// NewBitPackedEratosthenes creates a bit-packed segmented sieve.
func NewBitPackedEratosthenes(limit, wheelMod uint64) *BitPackedEratosthenes {
	w := NewWheel(wheelMod)
	ws := uint64((w.SpokeCount + 63) / 64)
	return &BitPackedEratosthenes{
		limit:      limit,
		segSpan:    (262144 / uint64(w.SpokeCount)) * w.Modulus,
		wheel:      w,
		wordStride: ws,
	}
}

// ForEachPrime calls fn for each prime up to the limit.
// Returns early if fn returns false.
func (e *BitPackedEratosthenes) ForEachPrime(fn func(uint64) bool) {
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
