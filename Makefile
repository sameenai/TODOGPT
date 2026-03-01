.PHONY: all build run briefing server clean deps test test-verbose test-coverage lint vet

all: deps build

deps:
	go mod tidy

build:
	go build -o bin/briefing ./cmd/briefing
	go build -o bin/server ./cmd/server

run: build
	./bin/server

briefing: build
	./bin/briefing

server: build
	./bin/server

test:
	go test -race -count=1 -timeout 120s ./...

test-verbose:
	go test -v -race -count=1 -timeout 120s ./...

test-coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "HTML report: go tool cover -html=coverage.out"

lint: vet
	@echo "Checking formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Run gofmt on:" && gofmt -l . && exit 1)

vet:
	go vet ./...

ci: lint test build

clean:
	rm -rf bin/ coverage.out

init: build
	./bin/server --init
