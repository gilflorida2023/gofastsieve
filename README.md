# gofastsieve — Segmented Sieve with Math-KAT Bridge

[![Math-KAT Verified](https://img.shields.io/badge/Math--KAT-Verified-blueviolet)](https://github.com/gilflorida2023/math-kat)

Go port of [fastsieve](https://github.com/gilflorida2023/fastsieve) — a segmented prime sieve with integrated [Math-KAT](https://github.com/gilflorida2023/math-kat) streaming SHA-256 verification.

## Quick Start

```bash
go build -o fastsieve ./cmd/fastsieve
./fastsieve --count=1000 --hash
# 18ac898998c81cb9eb52d37be6cd452a3b19babedbdd5cc6e8ffff20e7c2b048
```

## Math-KAT Cross-Verification

Every gofastsieve hash matches the [Math-KAT](https://github.com/gilflorida2023/math-kat) manifest and is independently reproducible:

```bash
# Hash first 1000 primes
./fastsieve --count=1000 --hash

# Verify against expected hash
./fastsieve --count=1000 --verify=18ac898998c81cb9eb52d37be6cd452a3b19babedbdd5cc6e8ffff20e7c2b048
# OK: hash matches ...

# Pipe to math-kat CLI
python3 -m math_kat verify --sequence A000040 --manifest manifests/A000040.json --tier ci_smoke --stdin \
  < <(./fastsieve --count=1000 --output)
```

## Usage

```
Usage: fastsieve [flags]

Flags:
  --count uint     Generate first COUNT primes
  --limit uint     Generate primes up to LIMIT (value bound)
  --hash           Output Math-KAT SHA-256 hex hash
  --verify string  Verify hash against EXPECTED hex string
  --seg-size uint  Segment size in bytes (0 = auto)
  --output         Print integers to stdout
```

## Development

```bash
make test       # go test -v -race ./...
make bench      # go test -bench=. -benchmem ./...
make vet        # go vet ./...
```

## Related

- [math-kat](https://github.com/gilflorida2023/math-kat) — Known Answer Tests for OEIS integer sequences
- [fastsieve](https://github.com/gilflorida2023/fastsieve) — C wheel-210 reference implementation

## License

MIT
# gofastsieve
