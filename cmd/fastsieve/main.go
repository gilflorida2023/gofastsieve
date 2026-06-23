package main

import (
	"bufio"
	"crypto/sha256"
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/gilflorida2023/gofastsieve/internal/sieve"
)

func main() {
	count := flag.Uint64("count", 0, "Generate first COUNT primes")
	limit := flag.Uint64("limit", 0, "Generate primes up to LIMIT")
	wheelMod := flag.Uint64("wheel", 210, "Wheel modulus (2, 6, 30, 210, 2310)")
	hashFlag := flag.Bool("hash", false, "Output Math-KAT SHA-256 hex hash to stdout")
	verify := flag.String("verify", "", "Verify hash against EXPECTED hex string")
	output := flag.Bool("output", false, "Print integers to stdout")
	outputFile := flag.String("o", "", "Write primes to FILE (use - for stdout)")
	hashOutput := flag.String("hash-output", "", "Write primes to FILE and hash to FILE.sha256")
	verifyHash := flag.String("verify-hash", "", "Verify primes FILE against FILE.sha256")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: fastsieve [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Generate primes using a segmented sieve with Math-KAT hash bridge.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  fastsieve --count=100 --hash\n")
		fmt.Fprintf(os.Stderr, "  fastsieve --limit=16000000 --hash\n")
		fmt.Fprintf(os.Stderr, "  fastsieve --count=1000 --output\n")
		fmt.Fprintf(os.Stderr, "  fastsieve --wheel=30 --count=100 --hash\n")
		fmt.Fprintf(os.Stderr, "  fastsieve --count=1000 --o primes.txt\n")
		fmt.Fprintf(os.Stderr, "  fastsieve --hash-output primes.txt --limit=16000000\n")
		fmt.Fprintf(os.Stderr, "  fastsieve --verify-hash primes.txt\n")
		fmt.Fprintf(os.Stderr, "\nMath-KAT: https://github.com/gilflorida2023/math-kat\n")
	}
	flag.Parse()

	// --verify-hash mode: standalone verification of a primes file
	if *verifyHash != "" {
		verifyHashFile(*verifyHash)
		return
	}

	if *count == 0 && *limit == 0 {
		flag.Usage()
		os.Exit(1)
	}
	if *count > 0 && *limit > 0 {
		fmt.Fprintln(os.Stderr, "error: specify --count or --limit, not both")
		os.Exit(1)
	}
	if *count > 0 {
		runCount(*count, *wheelMod, *hashFlag, *output, *verify, *outputFile, *hashOutput)
	} else {
		runLimit(*limit, *wheelMod, *hashFlag, *output, *verify, *outputFile, *hashOutput)
	}
}

func runCount(count, wheelMod uint64, wantHash, output bool, expectedHex, outputFile, hashOutput string) {
	if count == 0 {
		return
	}
	limit := estimateNthPrime(count)
	s := newSieve(limit, wheelMod)
	runSieve(s, count, 0, wantHash, output, expectedHex, outputFile, hashOutput)
}

func runLimit(limit, wheelMod uint64, wantHash, output bool, expectedHex, outputFile, hashOutput string) {
	s := newSieve(limit, wheelMod)
	runSieve(s, 0, limit, wantHash, output, expectedHex, outputFile, hashOutput)
}

func newSieve(limit, wheelMod uint64) *sieve.Eratosthenes {
	return sieve.NewEratosthenesWithWheel(limit, wheelMod)
}

func runSieve(s *sieve.Eratosthenes, maxCount, maxLimit uint64, wantHash, output bool, expectedHex, outputFile, hashOutput string) {
	usingHashOutput := hashOutput != ""
	usingOutputFile := outputFile != ""

	// Determine modes
	computeHash := wantHash || expectedHex != "" || usingHashOutput
	if !computeHash && !output && !usingOutputFile && !usingHashOutput && expectedHex == "" {
		// Default: show hash
		computeHash = true
		wantHash = true
	}

	// Open output file if specified
	var outWriter *bufio.Writer
	var outFile *os.File
	if usingOutputFile || usingHashOutput {
		outPath := outputFile
		if usingHashOutput {
			outPath = hashOutput
		}
		var err error
		outFile, err = os.Create(outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer outFile.Close()
		outWriter = bufio.NewWriter(outFile)
		defer outWriter.Flush()
	}

	// Set up hasher
	var hasher *sieve.StreamHasher
	if computeHash {
		hasher = sieve.NewStreamHasher()
	}

	count := uint64(0)
	s.ForEachPrime(func(p uint64) bool {
		if maxCount > 0 && count >= maxCount {
			return false
		}
		if output {
			fmt.Println(p)
		}
		if outWriter != nil {
			fmt.Fprintf(outWriter, "%d\n", p)
		}
		if hasher != nil {
			hasher.WriteInt(p)
		}
		count++
		return true
	})

	// Write hash sidecar if --hash-output was used
	if usingHashOutput && hasher != nil {
		sidecarPath := hashOutput + ".sha256"
		sidecar := fmt.Sprintf("%x  %s\n", hasher.Sum(), hashOutput)
		if err := os.WriteFile(sidecarPath, []byte(sidecar), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	// Verify or output hash
	if hasher != nil {
		got := hasher.HexSum()
		if expectedHex != "" {
			if got == expectedHex {
				fmt.Fprintf(os.Stderr, "OK: hash matches %s\n", expectedHex)
			} else {
				fmt.Fprintf(os.Stderr, "FAIL: got %s, expected %s\n", got, expectedHex)
				os.Exit(2)
			}
		} else if wantHash {
			fmt.Println(got)
		}
	}
}

// verifyHashFile reads a primes file and verifies it against its .sha256 sidecar.
func verifyHashFile(path string) {
	got, err := fileHash(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	sidecarPath := path + ".sha256"
	data, err := os.ReadFile(sidecarPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var expected string
	if _, err := fmt.Sscanf(string(data), "%s", &expected); err != nil {
		fmt.Fprintf(os.Stderr, "error: parsing %s: %v\n", sidecarPath, err)
		os.Exit(1)
	}

	if got == expected {
		fmt.Fprintf(os.Stderr, "OK: hash matches %s\n", expected)
	} else {
		fmt.Fprintf(os.Stderr, "FAIL: got %s, expected %s\n", got, expected)
		os.Exit(2)
	}
}

// estimateNthPrime provides an upper bound for the nth prime.
func estimateNthPrime(n uint64) uint64 {
	if n < 6 {
		return 15
	}
	fn := float64(n)
	ln := math.Log(fn)
	bound := fn * (ln + math.Log(ln))
	return uint64(bound * 12 / 10)
}

// fileHash computes SHA-256 of a text file with one integer per line.
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if _, err := fmt.Fprintf(h, "%s\n", line); err != nil {
			return "", err
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
