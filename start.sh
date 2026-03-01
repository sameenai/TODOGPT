#!/bin/bash
# Daily Briefing Dashboard — Quick Start
#
# Usage:
#   ./start.sh              Start the dashboard server
#   ./start.sh briefing     Print terminal briefing only
#   ./start.sh init         Generate default config file
#   ./start.sh --port 9090  Start on a custom port

set -e

cd "$(dirname "$0")"

echo "Building Daily Briefing..."
go build -o bin/briefing ./cmd/briefing
go build -o bin/server ./cmd/server

case "${1:-server}" in
    briefing)
        echo ""
        ./bin/briefing
        ;;
    init)
        ./bin/server --init
        ;;
    server|*)
        shift 2>/dev/null || true
        ./bin/server "$@"
        ;;
esac
