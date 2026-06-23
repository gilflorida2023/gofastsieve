package sieve

// Wheel encapsulates all parameters for a wheel factorization modulus.
type Wheel struct {
	Modulus     uint64
	SpokeCount  int
	Spokes      []uint64
	ResidueToBit []int
	WheelPrimes []uint64
}

// DefaultWheel210 returns the default wheel-210 factorization (coprime to 2,3,5,7).
func DefaultWheel210() *Wheel {
	w := &Wheel{
		Modulus:    210,
		SpokeCount: 48,
		Spokes: []uint64{
			1, 11, 13, 17, 19, 23, 29, 31,
			37, 41, 43, 47, 53, 59, 61, 67,
			71, 73, 79, 83, 89, 97, 101, 103,
			107, 109, 113, 121, 127, 131, 137, 139,
			143, 149, 151, 157, 163, 167, 169, 173,
			179, 181, 187, 191, 193, 197, 199, 209,
		},
		WheelPrimes: []uint64{2, 3, 5, 7},
	}
	w.initResidueToBit()
	return w
}

func (w *Wheel) initResidueToBit() {
	w.ResidueToBit = make([]int, w.Modulus)
	for i := range w.ResidueToBit {
		w.ResidueToBit[i] = -1
	}
	for i, s := range w.Spokes {
		w.ResidueToBit[s] = i
	}
}

// NewWheel generates wheel parameters for the given modulus.
// Supported moduli: 2, 6, 30, 210, 2310 (products of consecutive primes starting from 2).
func NewWheel(mod uint64) *Wheel {
	primes := wheelPrimesForMod(mod)
	spokes := computeSpokes(mod, primes)
	w := &Wheel{
		Modulus:     mod,
		SpokeCount:  len(spokes),
		Spokes:      spokes,
		WheelPrimes: primes,
	}
	w.initResidueToBit()
	return w
}

func wheelPrimesForMod(mod uint64) []uint64 {
	switch mod {
	case 2:
		return []uint64{2}
	case 6:
		return []uint64{2, 3}
	case 30:
		return []uint64{2, 3, 5}
	case 210:
		return []uint64{2, 3, 5, 7}
	case 2310:
		return []uint64{2, 3, 5, 7, 11}
	case 30030:
		return []uint64{2, 3, 5, 7, 11, 13}
	default:
		return nil
	}
}

func computeSpokes(mod uint64, primes []uint64) []uint64 {
	var spokes []uint64
	for n := uint64(1); n < mod; n++ {
		coprime := true
		for _, p := range primes {
			if n%p == 0 {
				coprime = false
				break
			}
		}
		if coprime {
			spokes = append(spokes, n)
		}
	}
	return spokes
}

// WheelPrime holds data for marking multiples of a base prime across segments.
// The stepping algorithm uses separate block/residue tracking to avoid
// expensive modulo operations in the inner marking loop.
type WheelPrime struct {
	Prime   uint64
	BlkStep uint64 // Prime / WheelMod
	Step    uint64 // Prime % WheelMod
}

// NewWheelPrime creates a WheelPrime for prime p under the given wheel.
func NewWheelPrime(p uint64, w *Wheel) WheelPrime {
	return WheelPrime{
		Prime:   p,
		BlkStep: p / w.Modulus,
		Step:    p % w.Modulus,
	}
}
