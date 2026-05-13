.PHONY: build test lint cover clean

# frida-core-devkit paths — 发行前需安装到 /usr/local 或修改此处指向解压目录
FRIDA_DEVKIT ?= /tmp/frida-devkit
CGO_CFLAGS ?= -I$(FRIDA_DEVKIT)
CGO_LDFLAGS ?= -L$(FRIDA_DEVKIT)
export CGO_CFLAGS
export CGO_LDFLAGS

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
