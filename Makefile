BINARY := fastsieve
CMD := ./cmd/fastsieve
GOFLAGS := -ldflags="-s -w"

.PHONY: all build test bench clean

all: build

build:
	go build $(GOFLAGS) -o $(BINARY) $(CMD)

install:
	go install $(CMD)

test:
	go test -v -race ./...

bench:
	go test -bench=. -benchmem ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)

# Math-KAT verification targets
.PHONY: verify-100 verify-1000 verify-smoke verify-dev

verify-100: build
	./$(BINARY) --hash --limit=100

verify-1000: build
	./$(BINARY) --hash --limit=1000

verify-smoke: build
	./$(BINARY) --hash --limit=1000

verify-dev: build
	./$(BINARY) --hash --limit=100000

# Cross-verify against math-kat manifests (requires python math-kat CLI)
.PHONY: cross-verify

cross-verify:
	@echo "Cross-verify requires: pip install math-kat"
	python3 -m math_kat verify --sequence A000040 --manifest manifests/A000040.json --tier ci_smoke
	./$(BINARY) --hash --limit=1000
