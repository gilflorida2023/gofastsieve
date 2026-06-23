package sieve

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"math"
	"strconv"
)

// StreamHasher computes a Math-KAT compatible SHA-256 hash over a stream of integers.
//
// Format: each integer is written as ASCII decimal digits followed by LF,
// including the last term. This matches the ascii_integer_lf formatter in math-kat.
type StreamHasher struct {
	hash    hash.Hash
	buf     []byte
	written int
}

// NewStreamHasher creates a new StreamHasher.
func NewStreamHasher() *StreamHasher {
	return &StreamHasher{hash: sha256.New()}
}

// WriteInt writes a uint64 to the hash stream in Math-KAT format.
func (sh *StreamHasher) WriteInt(n uint64) error {
	sh.buf = strconv.AppendUint(sh.buf[:0], n, 10)
	sh.buf = append(sh.buf, '\n')
	_, err := sh.hash.Write(sh.buf)
	if err == nil {
		sh.written++
	}
	return err
}

// Sum returns the SHA-256 hash of all integers written so far.
func (sh *StreamHasher) Sum() []byte {
	return sh.hash.Sum(nil)
}

// HexSum returns the hex-encoded SHA-256 hash.
func (sh *StreamHasher) HexSum() string {
	return fmt.Sprintf("%x", sh.Sum())
}

// WriteAll reads all values from the channel and writes them to the hasher.
func (sh *StreamHasher) WriteAll(values <-chan uint64) error {
	for v := range values {
		if err := sh.WriteInt(v); err != nil {
			return err
		}
	}
	return nil
}

// HashN generates first n primes and returns Math-KAT SHA-256 hex hash.
func HashN(n uint64) (string, error) {
	limit := estimateNthPrime(n)
	sieve := NewEratosthenes(limit)
	hasher := NewStreamHasher()
	count := uint64(0)
	sieve.ForEachPrime(func(p uint64) bool {
		if count >= n {
			return false
		}
		hasher.WriteInt(p)
		count++
		return true
	})
	if count < n {
		return HashN(n * 12 / 10)
	}
	return hasher.HexSum(), nil
}

// estimateNthPrime provides an upper bound for the nth prime.
// Uses n * (log n + log log n) for n >= 6 (Rosser's theorem).
func estimateNthPrime(n uint64) uint64 {
	if n < 6 {
		return 15
	}
	ln := math.Log(float64(n))
	bound := float64(n) * (ln + math.Log(ln))
	return uint64(bound * 12 / 10)
}
