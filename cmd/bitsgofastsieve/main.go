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
	wheelMod := flag.Uint64("wheel", 210, "Wheel modulus (30, 210, 2310); use fastsieve for wheel 2/6")
	hashFlag := flag.Bool("hash", false, "Output Math-KAT SHA-256 hex hash to stdout")
	verify := flag.String("verify", "", "Verify hash against EXPECTED hex string")
	output := flag.Bool("output", false, "Print integers to stdout")
	outputFile := flag.String("o", "", "Write primes to FILE (use - for stdout)")
	hashOutput := flag.String("hash-output", "", "Write primes to FILE and hash to FILE.sha256")
	verifyHash := flag.String("verify-hash", "", "Verify primes FILE against FILE.sha256")
	countOnly := flag.Bool("c", false, "Count only (no hash, no output, faster)")
	saveState := flag.Bool("s", false, "Save state file for resume")
	resume := flag.Bool("R", false, "Resume from saved state file")
	report := flag.Bool("r", false, "Report from existing state file (no sieve)")
	progress := flag.Bool("progress", false, "Show segment progress")
	stateFile := flag.String("state", "primes_state.bin", "State file path")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: bitsgofastsieve [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nMath-KAT: https://github.com/gilflorida2023/math-kat\n")
	}
	flag.Parse()

	if *verifyHash != "" {
		verifyHashFile(*verifyHash)
		return
	}

	if *report {
		reportState(*stateFile)
		return
	}

	if *resume {
		resumeSieve(*stateFile, *hashFlag, *verify, *output, *outputFile, *progress)
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
	switch *wheelMod {
	case 30, 210, 2310:
	default:
		fmt.Fprintf(os.Stderr, "error: wheel %d not supported in bitsgofastsieve (use fastsieve for wheel 2/6; supported here: 30, 210, 2310)\n", *wheelMod)
		os.Exit(1)
	}
	if *count > 0 {
		runCount(*count, *wheelMod, *hashFlag, *verify, *outputFile, *hashOutput, *output, *countOnly, *saveState, *stateFile, *progress)
	} else {
		runLimit(*limit, *wheelMod, *hashFlag, *verify, *outputFile, *hashOutput, *output, *countOnly, *saveState, *stateFile, *progress)
	}
}

func runCount(count, wheelMod uint64, wantHash bool, expectedHex, outputFile, hashOutput string, output, countOnly, saveState bool, statePath string, showProgress bool) {
	if count == 0 {
		return
	}
	limit := estimateNthPrime(count)
	s := newSieve(limit, wheelMod)

	var stateWriter *sieve.StateWriter
	if saveState {
		var err error
		stateWriter, err = sieve.NewStateWriter(statePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer stateWriter.Close()
	}

	runSieve(s, count, 0, wheelMod, wantHash, output, countOnly, expectedHex, outputFile, hashOutput, showProgress, stateWriter, nil, 0)
}

func runLimit(limit, wheelMod uint64, wantHash bool, expectedHex, outputFile, hashOutput string, output, countOnly, saveState bool, statePath string, showProgress bool) {
	s := newSieve(limit, wheelMod)

	var stateWriter *sieve.StateWriter
	if saveState {
		var err error
		stateWriter, err = sieve.NewStateWriter(statePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer stateWriter.Close()
	}

	runSieve(s, 0, limit, wheelMod, wantHash, output, countOnly, expectedHex, outputFile, hashOutput, showProgress, stateWriter, nil, 0)
}

func newSieve(limit, wheelMod uint64) *sieve.BitPackedEratosthenes {
	return sieve.NewBitPackedEratosthenes(limit, wheelMod)
}

func runSieve(s *sieve.BitPackedEratosthenes, maxCount, maxLimit, wheelMod uint64, wantHash, output, countOnly bool, expectedHex, outPath, hashOutPath string, showProgress bool, stateWriter *sieve.StateWriter, hasher *sieve.StreamHasher, resumeCount uint64) {
	usingHashOutput := hashOutPath != ""
	usingOutputFile := outPath != ""

	if countOnly {
		wantHash = false
	}

	computeHash := (wantHash || expectedHex != "" || usingHashOutput) && hasher == nil
	if hasher == nil && !computeHash && !output && !usingOutputFile && !usingHashOutput && expectedHex == "" && !countOnly && stateWriter == nil {
		computeHash = true
		wantHash = true
	}
	if hasher == nil && computeHash {
		hasher = sieve.NewStreamHasher()
	}

	var outWriter *bufio.Writer
	var outFile *os.File
	if usingOutputFile || usingHashOutput {
		outPath2 := outPath
		if usingHashOutput {
			outPath2 = hashOutPath
		}
		var err error
		outFile, err = os.Create(outPath2)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer outFile.Close()
		outWriter = bufio.NewWriter(outFile)
		defer outWriter.Flush()
	}

	var progressReporter *sieve.ProgressReporter
	if showProgress && maxLimit > 11 {
		progressReporter = sieve.NewProgressReporter(maxLimit)
		fmt.Fprintf(os.Stderr, "sieving to %d using wheel-%d (bit-packed)\n", maxLimit, wheelMod)
	}

	count := uint64(0)
	s.ForEachPrime(func(p uint64) bool {
		if maxCount > 0 && count >= maxCount {
			return false
		}

		if count < resumeCount {
			count++
			return true
		}

		if output {
			fmt.Println(p)
		}
		if outWriter != nil {
			fmt.Fprintf(outWriter, "%d\n", p)
		}
		if stateWriter != nil {
			stateWriter.WritePrime(p)
		}
		if hasher != nil {
			hasher.WriteInt(p)
		}
		count++

		if stateWriter != nil && count%100000 == 0 {
			stateWriter.Checkpoint(wheelMod, maxLimit, p, count)
		}

		if progressReporter != nil {
			progressReporter.ReportPrime(p, count)
		}

		return true
	})

	if stateWriter != nil && count > 0 {
		stateWriter.Checkpoint(wheelMod, maxLimit, maxLimit, count)
	}

	if progressReporter != nil {
		progressReporter.Done(count)
	}

	if usingHashOutput && hasher != nil {
		sidecarPath := hashOutPath + ".sha256"
		sidecar := fmt.Sprintf("%x  %s\n", hasher.Sum(), hashOutPath)
		if err := os.WriteFile(sidecarPath, []byte(sidecar), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

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

func resumeSieve(statePath string, wantHash bool, expectedHex string, output bool, outPath string, showProgress bool) {
	header, err := sieve.ReadStateHeader(statePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "resuming: %d primes found up to %d, target %d\n",
		header.TotalPrimes, header.LastSieved, header.Target)

	if header.Target <= header.LastSieved {
		fmt.Fprintf(os.Stderr, "target already reached\n")
		return
	}

	hasher := sieve.NewStreamHasher()
	primeCount, err := sieve.ReadPrimesFromState(statePath, hasher)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading state primes: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "caught up: %d primes hashed from state file\n", primeCount)

	stateWriter, err := sieve.NewStateWriterAppend(statePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer stateWriter.Close()

	s := newSieve(header.Target, header.WheelMod)

	runSieve(s, 0, header.Target, header.WheelMod,
		wantHash, output, false, expectedHex, outPath, "",
		showProgress, stateWriter, hasher, primeCount)
}

func reportState(path string) {
	header, err := sieve.ReadStateHeader(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	primeCount := uint64(0)
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			if lineNum <= 3 {
				continue
			}
			if scanner.Text() != "" {
				primeCount++
			}
		}
	}

	fmt.Printf("State file: %s\n", path)
	fmt.Printf("Wheel modulus: %d\n", header.WheelMod)
	fmt.Printf("Target: %d\n", header.Target)
	fmt.Printf("Last sieved: %d\n", header.LastSieved)
	fmt.Printf("Primes found: %d\n", header.TotalPrimes)
	fmt.Printf("Primes in file: %d\n", primeCount)
	progress := float64(header.LastSieved) / float64(header.Target) * 100
	fmt.Printf("Progress: %.2f%%\n", progress)
}

func estimateNthPrime(n uint64) uint64 {
	if n < 6 {
		return 15
	}
	fn := float64(n)
	ln := math.Log(fn)
	bound := fn * (ln + math.Log(ln))
	return uint64(bound * 12 / 10)
}

func verifyHashFile(path string) {
	got, err := hashFile(path)
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

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			continue
		}
		if _, err := fmt.Fprintf(h, "%s\n", text); err != nil {
			return "", err
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
