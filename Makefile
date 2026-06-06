# OrvixPanel — Makefile
# Common targets:
#   make build           — build the Go binary (requires `frontend/dist` to exist)
#   make build-frontend  — build the React frontend
#   make build-all       — build all targets (linux/amd64 + linux/arm64)
#   make dev             — run from source in dev mode
#   make test            — go test ./... with race + coverage
#   make lint            — golangci-lint
#   make clean           — remove build artifacts

VERSION    := $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS    := -ldflags="-s -w -X main.version=$(VERSION)"
TARGETS    := linux/amd64 linux/arm64
BIN_DIR    := bin

.PHONY: all
all: build

.PHONY: build-frontend
build-frontend:
	cd frontend && npm ci --no-audit --no-fund
	cd frontend && npm run build

.PHONY: build
build: build-frontend
	mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/orvixpanel ./cmd/orvixpanel

.PHONY: build-all
build-all: build-frontend
	@mkdir -p $(BIN_DIR)
	@for target in $(TARGETS); do \
		GOOS=$$(echo $$target | cut -d/ -f1) \
		GOARCH=$$(echo $$target | cut -d/ -f2) \
		go build $(LDFLAGS) -o $(BIN_DIR)/orvixpanel-$$(echo $$target | tr / -) ./cmd/orvixpanel; \
	done
	@for binary in $(BIN_DIR)/orvixpanel-*; do \
		sha256sum $$binary > $$binary.sha256; \
	done

.PHONY: dev
dev:
	ORVIX_ALLOW_DEV=1 ORVIX_SERVER_DEBUG=1 \
		go run ./cmd/orvixpanel

.PHONY: test
test:
	go test ./... -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: clean
clean:
	rm -rf $(BIN_DIR) coverage.out coverage.html
	rm -rf frontend/dist
	rm -rf internal/embed/dist

.PHONY: tidy
tidy:
	go mod tidy
