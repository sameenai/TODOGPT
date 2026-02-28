.PHONY: all build run briefing server clean deps

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

clean:
	rm -rf bin/

init:
	./bin/server --init
