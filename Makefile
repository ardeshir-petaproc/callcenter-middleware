.PHONY: build test run check-loc clean

BINARY=bin/middleware

build:
	go build -o $(BINARY) ./cmd/middleware

test:
	go test -v -race ./...

run: build
	./$(BINARY)

check-loc:
	@find . -name "*.go" -not -path "./vendor/*" | while read -r file; do \
		lines=$$(wc -l < "$$file"); \
		if [ "$$lines" -gt 180 ]; then \
			echo "LOC VIOLATION: $$file has $$lines lines (max 180)"; \
			exit 1; \
		fi; \
	done
	@echo "All Go files <= 180 LOC. Check passed."

clean:
	rm -rf bin/
