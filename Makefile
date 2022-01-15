PKGS = $(shell go list ./... | grep -v /vendor/)
GOFILES = $(shell find . -name '*.go' -and -not -path "./vendor/*")
UNFMT = $(shell gofmt -l ${GOFILES})
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GIT_COMMIT = $(shell git rev-parse --short HEAD)
LD_FLAGS = -s -w -extldflags '-static'

test: fmtcheck
	@echo "==> Running tests"
	@go test -v -count=1 -timeout=300s ${PKGS}

.PHONY: test

fmt:
	@echo "==> Fixing code with gofmt"
	@gofmt -s -w ${GOFILES}

.PHONY: fmt

fmtcheck:
	@echo "==> Checking code for gofmt compliance"
	@[ -z "${UNFMT}" ] || ( echo "Following files are not gofmt compliant.\n\n${UNFMT}\n\nRun 'make fmt' for reformat code"; exit 1 )

.PHONY: fmtcheck

lint: fmtcheck
	@golangci-lint run

.PHONY: lint

build:
	@echo "${GIT_COMMIT}" > version.txt
	CGO_ENABLED="0" go build -a -o="pkg/simple-proxy" -ldflags "${LD_FLAGS}"
.PHONY: build

doc:
	@docker run --rm -it -v $(PWD):/docs squidfunk/mkdocs-material build
.PHONY: doc

doc-server:
	@docker run --rm -it -p 8000:8000 -v $(PWD):/docs squidfunk/mkdocs-material serve --dev-addr 0.0.0.0:8000
.PHONY: doc-server