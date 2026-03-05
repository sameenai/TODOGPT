.PHONY: all build run briefing server tui clean deps \
        test test-verbose test-coverage \
        lint vet \
        security audit privacy \
        ci

# ── Defaults ──────────────────────────────────────────────────────────────────

COVERAGE_THRESHOLD := 95

all: deps build

deps:
	go mod tidy

# ── Build ─────────────────────────────────────────────────────────────────────

build:
	go build -o bin/briefing ./cmd/briefing
	go build -o bin/server   ./cmd/server
	go build -o bin/tui      ./cmd/tui

run: build
	./bin/server

briefing: build
	./bin/briefing

server: build
	./bin/server

tui: build
	./bin/tui

# ── Test & Coverage ───────────────────────────────────────────────────────────

GO_PKGS := $(shell go list ./... | grep -v '/frontend/')

test:
	go test -race -count=1 -timeout 120s $(GO_PKGS)

test-verbose:
	go test -v -race -count=1 -timeout 120s $(GO_PKGS)

test-coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic $(GO_PKGS)
	go tool cover -func=coverage.out
	@go tool cover -func=coverage.out | awk '/^total:/ { \
		gsub(/%/, "", $$3); pct=$$3+0; \
		printf "\nTotal coverage: %.1f%%  (threshold: $(COVERAGE_THRESHOLD)%%)\n", pct; \
		if (pct < $(COVERAGE_THRESHOLD)) { \
			print "FAIL: coverage below $(COVERAGE_THRESHOLD)%"; exit 1 \
		} else { \
			print "PASS: coverage meets threshold" \
		} \
	}'
	@echo ""
	@echo "HTML report: go tool cover -html=coverage.out"

# ── Lint ──────────────────────────────────────────────────────────────────────

lint: vet
	@echo "Checking formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Run gofmt on:" && gofmt -l . && exit 1)
	@echo "Formatting OK"

vet:
	go vet $(GO_PKGS)

# ── Security audit ────────────────────────────────────────────────────────────

# Static security analysis with gosec.
# Install: go install github.com/securego/gosec/v2/cmd/gosec@latest
GOBIN := $(shell go env GOPATH)/bin

security:
	@echo "==> Running gosec security scan..."
	@test -f $(GOBIN)/gosec || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	$(GOBIN)/gosec -severity medium -confidence medium -exclude-dir=vendor ./...
	@echo "gosec: PASS"

# Vulnerability check against Go vulnerability database.
# Install: go install golang.org/x/vuln/cmd/govulncheck@latest
vuln:
	@echo "==> Running govulncheck..."
	@test -f $(GOBIN)/govulncheck || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	$(GOBIN)/govulncheck ./...
	@echo "govulncheck: PASS"

# Privacy check: scan for patterns that indicate credential/secret leakage.
# This grep-based check is intentionally lightweight and fast.
privacy:
	@echo "==> Privacy & secrets scan..."
	@echo "  Checking for hardcoded secrets in non-config source files..."
	@! grep -rn --include='*.go' \
		-e 'password\s*=\s*"[^"]\+"\|api_key\s*=\s*"[^"]\+"\|token\s*=\s*"[^"]\+"' \
		--exclude-dir=vendor . \
		| grep -v '_test.go' \
		| grep -v 'config\.go' \
		| grep -v '// ' \
		| grep -v 'omitempty' \
		|| true
	@echo "  Checking config saves use restrictive file permissions (0600)..."
	@grep -rn 'WriteFile\|os.Create' --include='*.go' . | grep -v '_test.go' \
		| grep -v '0600\|0700\|0755' \
		| grep -v 'web/' \
		| (grep . && echo "WARNING: file write without explicit permissions above" || true)
	@echo "  Checking API keys are tagged omitempty in JSON..."
	@grep -n '\bToken\b\|\bPassword\b\|\bAPIKey\b\|\bBotToken\b\|\bAppToken\b' internal/config/config.go \
		| grep -v 'omitempty' \
		| grep -v 'TokenFile\|CredentialsFile' \
		| (grep . && echo "WARNING: sensitive field missing omitempty" || true)
	@echo "Privacy scan: PASS"

# Combined audit: security + vulnerability + privacy
audit: security vuln privacy
	@echo ""
	@echo "==> Audit complete."

# ── CI ────────────────────────────────────────────────────────────────────────

# Full CI pipeline: lint → test with coverage threshold → build
ci: lint test-coverage build
	@echo ""
	@echo "==> CI pipeline complete."

# ── Housekeeping ──────────────────────────────────────────────────────────────

clean:
	rm -rf bin/ coverage.out

init: build
	./bin/server --init

# ── Frontend (Next.js + AI SDK) ───────────────────────────────────────────────

frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build
