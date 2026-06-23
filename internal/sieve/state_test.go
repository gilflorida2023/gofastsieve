package sieve

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"testing"
)

func TestStateWriterRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.bin")

	sw, err := NewStateWriter(path)
	if err != nil {
		t.Fatalf("NewStateWriter: %v", err)
	}

	primes := []uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29}
	for _, p := range primes {
		if err := sw.WritePrime(p); err != nil {
			t.Fatalf("WritePrime(%d): %v", p, err)
		}
	}

	if err := sw.Checkpoint(210, 100, 29, 10); err != nil {
		t.Fatalf("Checkpoint: %v", err)
	}
	if err := sw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	header, err := ReadStateHeader(path)
	if err != nil {
		t.Fatalf("ReadStateHeader: %v", err)
	}

	if header.Magic != stateMagic {
		t.Errorf("Magic = 0x%08X, want 0x%08X", header.Magic, stateMagic)
	}
	if header.Version != stateVersion {
		t.Errorf("Version = %d, want %d", header.Version, stateVersion)
	}
	if header.WheelMod != 210 {
		t.Errorf("WheelMod = %d, want 210", header.WheelMod)
	}
	if header.Target != 100 {
		t.Errorf("Target = %d, want 100", header.Target)
	}
	if header.LastSieved != 29 {
		t.Errorf("LastSieved = %d, want 29", header.LastSieved)
	}
	if header.TotalPrimes != 10 {
		t.Errorf("TotalPrimes = %d, want 10", header.TotalPrimes)
	}

	hasher := NewStreamHasher()
	count, err := ReadPrimesFromState(path, hasher)
	if err != nil {
		t.Fatalf("ReadPrimesFromState: %v", err)
	}
	if count != 10 {
		t.Errorf("ReadPrimesFromState returned %d primes, want 10", count)
	}

	h := sha256.New()
	for _, p := range primes {
		fmt.Fprintf(h, "%d\n", p)
	}
	expected := h.Sum(nil)
	got := hasher.Sum()
	if string(got) != string(expected) {
		t.Errorf("hash mismatch: got %x, want %x", got, expected)
	}
}

func TestStateWriterAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.bin")

	sw, err := NewStateWriter(path)
	if err != nil {
		t.Fatalf("NewStateWriter: %v", err)
	}
	for _, p := range []uint64{2, 3, 5} {
		sw.WritePrime(p)
	}
	sw.Checkpoint(210, 20, 5, 3)
	sw.Close()

	sw2, err := NewStateWriterAppend(path)
	if err != nil {
		t.Fatalf("NewStateWriterAppend: %v", err)
	}
	for _, p := range []uint64{7, 11, 13} {
		sw2.WritePrime(p)
	}
	sw2.Checkpoint(210, 20, 13, 6)
	sw2.Close()

	hasher := NewStreamHasher()
	count, err := ReadPrimesFromState(path, hasher)
	if err != nil {
		t.Fatalf("ReadPrimesFromState: %v", err)
	}
	if count != 6 {
		t.Errorf("got %d primes, want 6", count)
	}

	h := sha256.New()
	for _, p := range []uint64{2, 3, 5, 7, 11, 13} {
		fmt.Fprintf(h, "%d\n", p)
	}
	if string(hasher.Sum()) != string(h.Sum(nil)) {
		t.Error("hash mismatch after append")
	}

	header, err := ReadStateHeader(path)
	if err != nil {
		t.Fatalf("ReadStateHeader: %v", err)
	}
	if header.LastSieved != 13 {
		t.Errorf("LastSieved = %d, want 13", header.LastSieved)
	}
	if header.TotalPrimes != 6 {
		t.Errorf("TotalPrimes = %d, want 6", header.TotalPrimes)
	}
}

func TestReadStateHeaderInvalid(t *testing.T) {
	dir := t.TempDir()

	t.Run("bad magic", func(t *testing.T) {
		path := filepath.Join(dir, "bad_magic.bin")
		os.WriteFile(path, make([]byte, headerSize), 0644)
		_, err := ReadStateHeader(path)
		if err == nil {
			t.Error("expected error for bad magic, got nil")
		}
	})

	t.Run("bad version", func(t *testing.T) {
		path := filepath.Join(dir, "bad_version.bin")
		header := make([]byte, headerSize)
		binary.LittleEndian.PutUint32(header[0:4], stateMagic)
		binary.LittleEndian.PutUint32(header[4:8], 999)
		os.WriteFile(path, header, 0644)
		_, err := ReadStateHeader(path)
		if err == nil {
			t.Error("expected error for bad version, got nil")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := ReadStateHeader(filepath.Join(dir, "nonexistent.bin"))
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})
}

func TestReadPrimesFromStateEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.bin")

	sw, err := NewStateWriter(path)
	if err != nil {
		t.Fatalf("NewStateWriter: %v", err)
	}
	sw.Close()

	hasher := NewStreamHasher()
	count, err := ReadPrimesFromState(path, hasher)
	if err != nil {
		t.Fatalf("ReadPrimesFromState: %v", err)
	}
	if count != 0 {
		t.Errorf("got %d primes from empty file, want 0", count)
	}
}

func TestEstimatePi(t *testing.T) {
	cases := []struct {
		x     uint64
		check func(uint64)
	}{
		{0, func(got uint64) {
			if got != 0 {
				t.Errorf("estimatePi(0) = %d, want 0", got)
			}
		}},
		{1, func(got uint64) {
			if got != 0 {
				t.Errorf("estimatePi(1) = %d, want 0", got)
			}
		}},
		{1000000, func(got uint64) {
			if got < 70000 || got > 80000 {
				t.Errorf("estimatePi(1000000) = %d, expected ~78498", got)
			}
		}},
	}
	for _, c := range cases {
		c.check(estimatePi(c.x))
	}
}

func TestRoundDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want time.Duration
	}{
		{500 * time.Nanosecond, time.Microsecond},        // rounds to nearest µs
		{1500 * time.Nanosecond, 2 * time.Microsecond},   // rounds to nearest µs
		{50 * time.Millisecond, 50 * time.Millisecond},   // < 1s → µs precision, exact
		{1500 * time.Millisecond, 1500 * time.Millisecond}, // 1.5s → ms precision, exact
		{90 * time.Minute, 90 * time.Minute},              // >= 1min → second precision, exact
	}
	for _, c := range cases {
		got := roundDuration(c.d)
		if got != c.want {
			t.Errorf("roundDuration(%v) = %v, want %v", c.d, got, c.want)
		}
	}
}
