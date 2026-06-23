package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/gilflorida2023/gofastsieve/internal/sieve"
)

func main() {
	count := flag.Uint64("count", 0, "Generate first COUNT primes")
	limit := flag.Uint64("limit", 0, "Generate primes up to LIMIT (value bound)")
	hashFlag := flag.Bool("hash", false, "Output Math-KAT SHA-256 hex hash")
	verify := flag.String("verify", "", "Verify hash against EXPECTED hex string")
	segSize := flag.Uint64("seg-size", 0, "Segment size in bytes (0 = auto)")
	output := flag.Bool("output", false, "Print integers to stdout")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: fastsieve [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Generate primes using a segmented sieve with Math-KAT hash bridge.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  fastsieve --count=100 --hash\n")
		fmt.Fprintf(os.Stderr, "  fastsieve --limit=1000000 --hash --verify=<expected>\n")
		fmt.Fprintf(os.Stderr, "  fastsieve --count=1000 --output\n")
		fmt.Fprintf(os.Stderr, "\nMath-KAT: https://github.com/gilflorida2023/math-kat\n")
	}
	flag.Parse()

	if *count == 0 && *limit == 0 {
		flag.Usage()
		os.Exit(1)
	}
	if *count > 0 && *limit > 0 {
		fmt.Fprintln(os.Stderr, "error: specify --count or --limit, not both")
		os.Exit(1)
	}
	_ = segSize // available for future segment size tuning

	if *count > 0 {
		runCount(*count, *hashFlag, *verify, *output)
	} else {
		runLimit(*limit, *hashFlag, *verify, *output)
	}
}

func runCount(count uint64, wantHash bool, expectedHex string, output bool) {
	if count == 0 {
		return
	}
	limit := estimateNthPrime(count)
	s := sieve.NewEratosthenes(limit)

	var hasher *sieve.StreamHasher
	if wantHash || expectedHex != "" || (!output && expectedHex == "") {
		hasher = sieve.NewStreamHasher()
		if !wantHash && expectedHex == "" {
			wantHash = true
		}
	}

	n := uint64(0)
	s.ForEachPrime(func(p uint64) bool {
		if n >= count {
			return false
		}
		if output {
			fmt.Println(p)
		}
		if hasher != nil {
			hasher.WriteInt(p)
		}
		n++
		return true
	})

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

func runLimit(limit uint64, wantHash bool, expectedHex string, output bool) {
	s := sieve.NewEratosthenes(limit)

	var hasher *sieve.StreamHasher
	if wantHash || expectedHex != "" || (!output && expectedHex == "") {
		hasher = sieve.NewStreamHasher()
		if !wantHash && expectedHex == "" {
			wantHash = true
		}
	}

	s.ForEachPrime(func(p uint64) bool {
		if output {
			fmt.Println(p)
		}
		if hasher != nil {
			hasher.WriteInt(p)
		}
		return true
	})

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

// estimateNthPrime provides an upper bound for the nth prime.
// Uses n * (log n + log log n) with 20% safety margin (Rosser's theorem).
func estimateNthPrime(n uint64) uint64 {
	if n < 6 {
		return 15
	}
	fn := float64(n)
	ln := math.Log(fn)
	bound := fn * (ln + math.Log(ln))
	return uint64(bound * 12 / 10)
}
