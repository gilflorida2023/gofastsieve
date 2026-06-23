package sieve

const (
	WheelMod   = 210
	WheelSpoke = 48
)

// Spokes lists residues modulo 210 coprime to 2,3,5,7.
// These 48 values are the only possible prime positions (candidates)
// after wheel factorization.
var Spokes = [WheelSpoke]uint64{
	1, 11, 13, 17, 19, 23, 29, 31,
	37, 41, 43, 47, 53, 59, 61, 67,
	71, 73, 79, 83, 89, 97, 101, 103,
	107, 109, 113, 121, 127, 131, 137, 139,
	143, 149, 151, 157, 163, 167, 169, 173,
	179, 181, 187, 191, 193, 197, 199, 209,
}

// ResidueToBit maps residue 0..209 → Spokes index (0..47) or -1.
var ResidueToBit [WheelMod]int8

func init() {
	for i := range ResidueToBit {
		ResidueToBit[i] = -1
	}
	for i, s := range Spokes {
		ResidueToBit[s] = int8(i)
	}
}

// WheelPrime holds data for marking multiples of a base prime across segments.
// The stepping algorithm uses separate block/residue tracking to avoid
// expensive modulo operations in the inner marking loop.
type WheelPrime struct {
	Prime   uint64
	BlkStep uint64 // Prime / WheelMod
	Step    uint64 // Prime % WheelMod
}

// NewWheelPrime creates a WheelPrime for prime p.
func NewWheelPrime(p uint64) WheelPrime {
	return WheelPrime{
		Prime:   p,
		BlkStep: p / WheelMod,
		Step:    p % WheelMod,
	}
}
