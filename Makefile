BINARY := mcpup
MAIN := ./cmd/mcpup

.PHONY: build test lint fmt run clean

build:
	go build -o bin/$(BINARY) $(MAIN)

test:
	go test ./...

lint:
	go vet ./...

fmt:
	go fmt ./...

run:
	go run $(MAIN)

clean:
	rm -rf bin
