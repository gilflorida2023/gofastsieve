package sieve

import (
	"math"
	"math/bits"
	"runtime"
	"sync/atomic"
)

const minParallelLimit = 1_000_000

func (e *BitPackedEratosthenes) parallelGenerate(emit func(uint64) bool) {
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
	if lo > e.limit {
		return
	}

	nCPU := runtime.GOMAXPROCS(0)
	if nCPU < 2 {
		e.generate(emit)
		return
	}

	totalSpan := e.limit - lo + 1
	if totalSpan < 100000 {
		e.generate(emit)
		return
	}

	var stop atomic.Bool
	var seg []uint64

	for lo <= e.limit {
		if stop.Load() {
			break
		}

		hi := lo + e.segSpan - 1
		if hi > e.limit || hi < lo {
			hi = e.limit
		}

		firstBlock := lo / w.Modulus
		lastBlock := hi / w.Modulus
		numBlocks := lastBlock - firstBlock + 1
		segLen := numBlocks * ws

		// Shared buffer: each goroutine writes to non-overlapping block regions.
		// Different core → different cache lines → no race on x86-64.
		if cap(seg) < int(segLen) {
			seg = make([]uint64, segLen)
		}
		seg = seg[:segLen]

		type blockRange struct{ start, end uint64 }
		ranges := make([]blockRange, nCPU)
		for i := 0; i < nCPU; i++ {
			br := blockRange{
				start: uint64(i) * numBlocks / uint64(nCPU),
				end:   uint64(i+1) * numBlocks / uint64(nCPU),
			}
			if i == nCPU-1 {
				br.end = numBlocks
			}
			ranges[i] = br
		}

		// Init buffer in parallel (each goroutine inits its own blocks)
		for i := 0; i < nCPU; i++ {
			br := ranges[i]
			off := br.start * ws
			nw := (br.end - br.start) * ws
			for j := off; j < off+nw; j++ {
				seg[j] = ^uint64(0)
			}
		}
		if lastBits := uint(w.SpokeCount % 64); lastBits > 0 && segLen > 0 {
			seg[segLen-1] &= (1 << lastBits) - 1
		}

		// Launch goroutines to process block ranges
		done := make(chan int, nCPU)
		for i := 0; i < nCPU; i++ {
			br := ranges[i]
			clo := firstBlock*w.Modulus + (br.start)*w.Modulus
			chi := firstBlock*w.Modulus + br.end*w.Modulus - 1
			if clo < lo {
				clo = lo
			}
			if chi > hi {
				chi = hi
			}
			if br.start >= br.end {
				done <- i
				continue
			}

			go func(gid int, startBlk, endBlk uint64, clo, chi uint64) {
				off := startBlk * ws
				buf := seg[off : off+(endBlk-startBlk)*ws]
				nb := endBlk - startBlk

				// Phase 0: pre-sieve small primes
				for _, psm := range e.preSieveMasks {
					p := psm.prime
					if p > sqrtLimit {
						continue
					}
					masks := psm.masks
					for bi := uint64(0); bi < nb; bi++ {
						bMod := (firstBlock + startBlk + bi) % p
						moff := bMod * ws
						off2 := bi * ws
						for wi := uint64(0); wi < ws; wi++ {
							buf[off2+wi] &^= masks[moff+wi]
						}
					}
				}

				// Phase 1: mark composites for all base primes
				for _, wp := range wheels {
					p := wp.Prime
					if p > hi {
						continue
					}

					m := clo
					if rem := m % p; rem != 0 {
						m += p - rem
					}
					if m < p*p {
						m = p * p
					}
					if m > chi {
						continue
					}

					block := m / w.Modulus
					r := m % w.Modulus

					for block <= lastBlock {
						if block >= firstBlock+endBlk {
							break
						}
						if block >= firstBlock+startBlk {
							if ri := w.ResidueToBit[r]; ri >= 0 {
								wordIdx := uint64(ri) / 64
								bitIdx := uint64(ri) % 64
								buf[(block-firstBlock-startBlk)*ws+wordIdx] &^= 1 << bitIdx
							}
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

				done <- gid
			}(i, br.start, br.end, clo, chi)
		}

		// Wait for all goroutines
		for i := 0; i < nCPU; i++ {
			<-done
		}

		if stop.Load() {
			break
		}

		// Phase 2: scan survivors and emit
		for bi := uint64(0); bi < numBlocks; bi++ {
			base := bi * ws
			blockBase := (firstBlock + bi) * w.Modulus
			for wi := uint64(0); wi < ws; wi++ {
				word := seg[base+wi]
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
						stop.Store(true)
						return
					}
				}
			}
		}

		lo = hi + 1
	}
}