.PHONY: all validate tidy vet lint security test test-cover build clean

BIN       := tp
CMD       := ./cmd/tp
COVER_OUT := coverage.out

all: validate lint test build

## ── Validation ────────────────────────────────────────────────────────

validate: tidy vet

tidy:
	go mod verify
	go mod tidy
	git diff --exit-code go.mod go.sum

vet:
	go vet ./...

## ── Linting ───────────────────────────────────────────────────────────

lint:
	golangci-lint run --timeout=5m ./...

## ── Security ──────────────────────────────────────────────────────────

security:
	govulncheck ./...

## ── Testing ───────────────────────────────────────────────────────────

test:
	go test -v -race -coverprofile=$(COVER_OUT) -covermode=atomic -timeout=10m ./...

test-cover: test
	go tool cover -func=$(COVER_OUT)

## ── Build ─────────────────────────────────────────────────────────────

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o $(BIN) $(CMD)

clean:
	rm -f $(BIN) $(COVER_OUT)
