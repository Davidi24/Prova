export GO111MODULE=on

SHELL=/bin/bash -o pipefail

PWD = $(shell pwd)
GO ?= go

.PHONY: all
all: format kupon_tls_server bundle_deployment

.PHONY: kupon_tls_server
kupon_tls_server:
	$(GO) build -v --race -mod=vendor -o bin/$(SERVICE) .

.PHONY: windows
windows: format build_windows bundle_deployment

build_windows:
	GOOS=windows GOARCH=amd64 $(GO) build -v -mod=vendor -o bin/$(SERVICE) .

.PHONY: linux
linux: format build_linux bundle_deployment

build_linux:
	GOOS=linux GOARCH=amd64 $(GO) build -v -mod=vendor -o bin/$(SERVICE) .

.PHONY: test
test:
	$(GO) vet ./...
	$(GO) test -v -failfast $(go list ./... ) --race ./... -coverprofile=/tmp/vscode-goBXZcM3/go-code-cover -covermode atomic

format:
	$(GO) mod tidy
	$(GO) mod vendor
	$(GO) fmt ./...


bundle_deployment:
	cp -R kuponServerCertificate/ bin/kuponServerCertificate
	cp config.yaml bin/config.yaml
	rm -f bin/kupon_tls_server.zip
	cd bin && zip kupon_tls_server.zip kupon_tls_server
