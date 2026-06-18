BINARY    := webhound
MODULE    := github.com/digitalohara/webhound
MAIN      := ./cmd/webhound
VERSION   := 1.0.0
GOFLAGS   := -trimpath
LDFLAGS   := -ldflags "-s -w -X main.version=$(VERSION)"

# ── Build ──────────────────────────────────────────────────────────────────────
.PHONY: build
build:
	go build $(GOFLAGS) $(LDFLAGS) -o $(BINARY) $(MAIN)

.PHONY: build-all
build-all:
	GOOS=linux   GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o dist/$(BINARY)-linux-amd64   $(MAIN)
	GOOS=linux   GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o dist/$(BINARY)-linux-arm64   $(MAIN)
	GOOS=darwin  GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64  $(MAIN)
	GOOS=darwin  GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64  $(MAIN)
	GOOS=windows GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe $(MAIN)

# ── Dependencies ───────────────────────────────────────────────────────────────
.PHONY: deps
deps:
	go mod tidy
	go mod download

# ── Test ───────────────────────────────────────────────────────────────────────
.PHONY: test
test:
	go test -v -race -count=1 ./tests/unit/...

.PHONY: test-integration
test-integration:
	go test -v -race -count=1 -timeout 60s ./tests/integration/...

.PHONY: test-all
test-all: test test-integration

.PHONY: coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ── Code Quality ───────────────────────────────────────────────────────────────
.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: fmt
fmt:
	gofmt -w -s .
	goimports -w .

# ── Install ────────────────────────────────────────────────────────────────────
.PHONY: install
install:
	go install $(GOFLAGS) $(LDFLAGS) $(MAIN)

# ── Clean ──────────────────────────────────────────────────────────────────────
.PHONY: clean
clean:
	rm -f $(BINARY)
	rm -rf dist/
	rm -f coverage.out coverage.html

# ── Run (dev shortcut) ─────────────────────────────────────────────────────────
.PHONY: run
run: build
	./$(BINARY) scan --url $(URL)

# ── Help ───────────────────────────────────────────────────────────────────────
.PHONY: help
help:
	@echo "WebHound build targets:"
	@echo "  make build          Build binary for current platform"
	@echo "  make build-all      Cross-compile for Linux/macOS/Windows"
	@echo "  make deps           Download and tidy dependencies"
	@echo "  make test           Run unit tests"
	@echo "  make test-all       Run all tests including integration"
	@echo "  make coverage       Generate HTML coverage report"
	@echo "  make install        Install to GOPATH/bin"
	@echo "  make clean          Remove build artefacts"
	@echo "  make run URL=...    Quick dev run (e.g. make run URL=https://example.com)"
