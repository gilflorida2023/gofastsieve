package sieve

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"time"
)

const (
	stateMagic    = 0x46535350 // "FSSP"
	stateVersion  = 1
	headerSize    = 48
)

// StateHeader stores sieve state for resume capability.
type StateHeader struct {
	Magic       uint32
	Version     uint32
	WheelMod    uint64
	Target      uint64
	LastSieved  uint64
	TotalPrimes uint64
	Reserved    [8]byte
}

// StateWriter manages writing sieve state to a file.
type StateWriter struct {
	file *os.File
	path string
}

// NewStateWriter creates a state writer that writes primes to statePath
// and maintains a checkpoint header.
func NewStateWriter(statePath string) (*StateWriter, error) {
	f, err := os.Create(statePath)
	if err != nil {
		return nil, fmt.Errorf("creating state file: %w", err)
	}
	header := make([]byte, headerSize)
	if _, err := f.Write(header); err != nil {
		f.Close()
		return nil, fmt.Errorf("writing header: %w", err)
	}
	return &StateWriter{file: f, path: statePath}, nil
}

// WritePrime writes a prime to the state file.
func (sw *StateWriter) WritePrime(p uint64) error {
	_, err := fmt.Fprintf(sw.file, "%d\n", p)
	return err
}

// Checkpoint updates the header with current progress.
func (sw *StateWriter) Checkpoint(wheelMod, target, lastSieved, totalPrimes uint64) error {
	h := StateHeader{
		Magic:       stateMagic,
		Version:     stateVersion,
		WheelMod:    wheelMod,
		Target:      target,
		LastSieved:  lastSieved,
		TotalPrimes: totalPrimes,
	}
	return sw.writeHeader(h)
}

func (sw *StateWriter) writeHeader(h StateHeader) error {
	buf := make([]byte, headerSize)
	binary.LittleEndian.PutUint32(buf[0:4], h.Magic)
	binary.LittleEndian.PutUint32(buf[4:8], h.Version)
	binary.LittleEndian.PutUint64(buf[8:16], h.WheelMod)
	binary.LittleEndian.PutUint64(buf[16:24], h.Target)
	binary.LittleEndian.PutUint64(buf[24:32], h.LastSieved)
	binary.LittleEndian.PutUint64(buf[32:40], h.TotalPrimes)
	if _, err := sw.file.WriteAt(buf, 0); err != nil {
		return fmt.Errorf("writing checkpoint: %w", err)
	}
	return nil
}

// NewStateWriterAppend opens an existing state file for appending.
// The file must already have a valid header at offset 0.
func NewStateWriterAppend(statePath string) (*StateWriter, error) {
	f, err := os.OpenFile(statePath, os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening state file: %w", err)
	}
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		f.Close()
		return nil, fmt.Errorf("seeking state file: %w", err)
	}
	return &StateWriter{file: f, path: statePath}, nil
}

// ReadPrimesFromState reads primes from a state file (skipping the binary header)
// and feeds each to the hasher. Returns the number of primes read.
func ReadPrimesFromState(path string, hasher *StreamHasher) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("opening state file: %w", err)
	}
	defer f.Close()

	// Skip the 48-byte binary header
	if _, err := f.Seek(headerSize, io.SeekStart); err != nil {
		return 0, fmt.Errorf("seeking past header: %w", err)
	}

	scanner := bufio.NewScanner(f)
	var count uint64
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		n, err := strconv.ParseUint(line, 10, 64)
		if err != nil {
			continue // skip any remaining header artifacts
		}
		if hasher != nil {
			if err := hasher.WriteInt(n); err != nil {
				return count, fmt.Errorf("hashing prime %d: %w", n, err)
			}
		}
		count++
	}
	return count, scanner.Err()
}

// Close closes the state file.
func (sw *StateWriter) Close() error {
	return sw.file.Close()
}

// ReadStateHeader reads the state header from a file.
func ReadStateHeader(path string) (*StateHeader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening state file: %w", err)
	}
	defer f.Close()

	buf := make([]byte, headerSize)
	if _, err := f.Read(buf); err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}

	h := &StateHeader{
		Magic:       binary.LittleEndian.Uint32(buf[0:4]),
		Version:     binary.LittleEndian.Uint32(buf[4:8]),
		WheelMod:    binary.LittleEndian.Uint64(buf[8:16]),
		Target:      binary.LittleEndian.Uint64(buf[16:24]),
		LastSieved:  binary.LittleEndian.Uint64(buf[24:32]),
		TotalPrimes: binary.LittleEndian.Uint64(buf[32:40]),
	}

	if h.Magic != stateMagic {
		return nil, fmt.Errorf("invalid state file: bad magic 0x%08X", h.Magic)
	}
	if h.Version > stateVersion {
		return nil, fmt.Errorf("unknown state version %d", h.Version)
	}

	return h, nil
}

// ProgressReporter prints progress based on prime value vs target.
type ProgressReporter struct {
	startTime time.Time
	lastTime  time.Time
	interval  uint64
	target    uint64
	nextCheck uint64
}

// NewProgressReporter creates a new progress reporter.
func NewProgressReporter(target uint64) *ProgressReporter {
	return &ProgressReporter{
		startTime: time.Now(),
		lastTime:  time.Now(),
		interval:  10000,
		target:    target,
		nextCheck: 10000,
	}
}

// ReportPrime is called for each prime found.
func (pr *ProgressReporter) ReportPrime(p uint64, count uint64) {
	if count >= pr.nextCheck {
		now := time.Now()
		elapsed := now.Sub(pr.startTime)
		pct := float64(p) / float64(pr.target) * 100
		rate := float64(count) / elapsed.Seconds()
		fmt.Fprintf(os.Stderr, "\r%d / ~%d primes (%.1f%%) | %d | %.0f primes/s    ",
			count, estimatePi(pr.target), pct, p, rate)
		pr.lastTime = now
		pr.nextCheck += pr.interval
	}
}

// Done prints final summary.
func (pr *ProgressReporter) Done(primes uint64) {
	elapsed := time.Since(pr.startTime)
	rate := float64(primes) / elapsed.Seconds()
	fmt.Fprintf(os.Stderr, "\rdone: %d primes in %v (%.0f primes/sec)\n",
		primes, roundDuration(elapsed), rate)
}

// estimatePi approximates the prime-counting function π(x).
func estimatePi(x uint64) uint64 {
	if x < 2 {
		return 0
	}
	fx := float64(x)
	return uint64(fx / (math.Log(fx) - 1.0))
}

func roundDuration(d time.Duration) time.Duration {
	if d < time.Second {
		return d.Round(time.Microsecond)
	}
	if d < time.Minute {
		return d.Round(time.Millisecond)
	}
	return d.Round(time.Second)
}
