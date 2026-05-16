.PHONY: build test lint cover clean devkit

# frida-core-devkit paths — 默认放在项目 .devkit/ (由 make devkit 下载)
FRIDA_DEVKIT ?= $(CURDIR)/.devkit
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

# 下载 frida-core-devkit（编译 fridaengine 必需）
devkit:
	@mkdir -p $(FRIDA_DEVKIT)
	@if [ ! -f $(FRIDA_DEVKIT)/libfrida-core.a ]; then \
		curl -sL "https://github.com/frida/frida/releases/download/17.9.8/frida-core-devkit-17.9.8-linux-x86_64.tar.xz" -o /tmp/frida-devkit.tar.xz && \
		tar -xf /tmp/frida-devkit.tar.xz -C $(FRIDA_DEVKIT)/ && \
		rm /tmp/frida-devkit.tar.xz && \
		echo "frida-core-devkit installed to $(FRIDA_DEVKIT)"; \
	else \
		echo "frida-core-devkit already installed at $(FRIDA_DEVKIT)"; \
	fi
