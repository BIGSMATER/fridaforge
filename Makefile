.PHONY: build test lint cover clean

build:
	go build -o fridaforge ./cmd/fridaforge/

test:
	go test -v ./...

lint:
	golangci-lint run ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "HTML report: go tool cover -html=coverage.out"

clean:
	rm -f fridaforge coverage.out
