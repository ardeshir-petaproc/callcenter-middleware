.PHONY: build web go test test-race run sim demo check-loc lint docker clean desktop e2e-asterisk

BINARY=bin/middleware
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS=-s -w -X callcenter-middleware/internal/server.Version=$(VERSION)

build: web go

# Build the React control panel into internal/web/dist (embedded in the binary).
web:
	cd web && npm ci && npm run build

go:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/middleware
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/ari-sim ./cmd/ari-sim

test:
	go test ./...

test-race:
	go test -race ./...

run: go
	./$(BINARY)

# Simulated Asterisk on :8088 generating inbound calls.
sim:
	go run ./cmd/ari-sim -listen :8088 -rate 4

# Full demo without a PBX: simulator + middleware with demo data on :8080.
demo: go
	./bin/ari-sim -listen :8088 -rate 4 & echo $$! > bin/.sim.pid
	SEED_DEMO=true DB_PATH=data/demo.db ./$(BINARY); kill `cat bin/.sim.pid`

check-loc:
	@find . -name "*.go" -not -path "./vendor/*" -not -path "./web/*" -not -path "./desktop/*" | while read -r file; do \
		lines=$$(wc -l < "$$file"); \
		if [ "$$lines" -gt 180 ]; then \
			echo "LOC VIOLATION: $$file has $$lines lines (max 180)"; \
			exit 1; \
		fi; \
	done
	@echo "All Go files <= 180 LOC. Check passed."

lint:
	go vet ./...
	cd web && npm run typecheck

docker:
	docker build --build-arg VERSION=$(VERSION) -t callcenter-middleware:$(VERSION) .

desktop:
	cd desktop && npm ci && npm run dist:linux

clean:
	rm -rf bin/

# Real Asterisk 18 (Docker) end-to-end scenarios with a fake OpenAI realtime server.
e2e-asterisk:
	cd test/asterisk-e2e && GOFLAGS=-mod=mod ./run.sh
