# Makefile for tffmt project

# Variables
PROJECT_NAME := tffmt
VERSION ?= $(shell git describe --tags --always || echo "0.1.0")
SHELL := /bin/bash

# Directories
BINDIR := bin
DISTDIR := dist
CMD_DIR := cmd/tffmt
PKG_DIR := pkg

# Go related variables
GO ?= go
GOFLAGS := -trimpath
LDFLAGS := -ldflags "-X github.com/krewenki/tffmt/cmd/tffmt.Version=$(VERSION) -w -s"

# Terraform related variables
TERRAFORM ?= terraform

# Default target
.PHONY: all
all: build

# Build the project
.PHONY: build
build:
	mkdir -p $(BINDIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINDIR)/$(PROJECT_NAME) .

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BINDIR) $(DISTDIR)
	$(GO) clean

# Install the binary
.PHONY: install
install: build
	install -d $(DESTDIR)/usr/local/bin
	install -m 755 $(BINDIR)/$(PROJECT_NAME) $(DESTDIR)/usr/local/bin/

# Run tests
.PHONY: test
test:
	$(GO) test -v ./...

# Generate test coverage
.PHONY: coverage
coverage:
	mkdir -p $(DISTDIR)
	$(GO) test -coverprofile=$(DISTDIR)/coverage.out ./...
	$(GO) tool cover -html=$(DISTDIR)/coverage.out -o $(DISTDIR)/coverage.html

# Format Go code
.PHONY: fmt
fmt:
	$(GO) fmt ./...

# Lint Go code
.PHONY: lint
lint:
	$(GO) vet ./...
	golangci-lint run --config .golangci.yml

# Terraform format checking
.PHONY: tf-fmt-check
tf-fmt-check:
	@find . -name "*.tf" -type f -not -path "*/\.*" -exec $(TERRAFORM) fmt -check=true -diff=true {} \;

# Format Terraform files
.PHONY: tf-fmt
tf-fmt:
	@find . -name "*.tf" -type f -not -path "*/\.*" -exec $(TERRAFORM) fmt {} \;

# Create a distribution package
.PHONY: dist
dist: clean build
	mkdir -p $(DISTDIR)
	cp $(BINDIR)/$(PROJECT_NAME) $(DISTDIR)/
	cp README.md LICENSE $(DISTDIR)/ 2>/dev/null || true
	cd $(DISTDIR) && tar -czf $(PROJECT_NAME)-$(VERSION).tar.gz *

# Cross-compilation for different platforms
.PHONY: dist-all
dist-all: clean
	mkdir -p $(DISTDIR)
	GOOS=linux GOARCH=amd64 $(MAKE) dist
	GOOS=darwin GOARCH=amd64 $(MAKE) dist
	GOOS=windows GOARCH=amd64 $(MAKE) dist

# Show version
.PHONY: version
version:
	@echo $(VERSION)

# Show help information
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all           - Build the project (default)"
	@echo "  build         - Build the binary"
	@echo "  clean         - Remove build artifacts"
	@echo "  install       - Install the binary"
	@echo "  test          - Run tests"
	@echo "  coverage      - Generate test coverage report"
	@echo "  fmt           - Format Go code"
	@echo "  lint          - Lint Go code"
	@echo "  tf-fmt-check  - Check Terraform formatting"
	@echo "  tf-fmt        - Format Terraform files"
	@echo "  dist          - Create a distribution package"
	@echo "  dist-all      - Create distribution for multiple platforms"
	@echo "  version       - Show version"
	@echo "  help          - Show this help message"
