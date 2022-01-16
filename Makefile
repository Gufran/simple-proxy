PKGS = $(shell go list ./... | grep -v /vendor/)
GOFILES = $(shell find . -name '*.go' -and -not -path "./vendor/*")
UNFMT = $(shell gofmt -l ${GOFILES})
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GIT_COMMIT = $(shell git rev-parse --short HEAD)
PROJECT := $(CURRENT_DIR:$(GOPATH)/src/%=%)
LD_FLAGS = -s -w -extldflags '-static'

XC_OS ?= darwin linux
XC_ARCH ?= amd64

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
	@CGO_ENABLED="0" GOOS=${GOOS} GOARCH=${GOARCH} go build -a -o="pkg/simple-proxy-${GOOS}_${GOARCH}" -ldflags "${LD_FLAGS}"
.PHONY: build

doc:
	@docker run --rm -it -v $(PWD):/docs squidfunk/mkdocs-material build
.PHONY: doc

doc-server:
	@docker run --rm -it -p 8000:8000 -v $(PWD):/docs squidfunk/mkdocs-material serve --dev-addr 0.0.0.0:8000
.PHONY: doc-server


define make-xc-target
  $1/$2:
		@printf "%s%20s %s\n" "-->" "${1}/${2}:" "${PROJECT}"
		@$(MAKE) build GOOS=$1 GOARCH=$2
  .PHONY: $1/$2

  $1:: $1/$2
  .PHONY: $1

  xc:: $1/$2
  .PHONY: xc
endef
$(foreach goarch,$(XC_ARCH),$(foreach goos,$(XC_OS),$(eval $(call make-xc-target,$(goos),$(goarch)))))