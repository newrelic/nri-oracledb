WORKDIR     := $(shell pwd)
TARGET      := target
TARGET_DIR   = $(WORKDIR)/$(TARGET)
NATIVEOS    := $(shell go version | awk -F '[ /]' '{print $$4}')
NATIVEARCH  := $(shell go version | awk -F '[ /]' '{print $$5}')
INTEGRATION := oracledb
BINARY_NAME  = nri-$(INTEGRATION)
GO_FILES    := ./src/
GOFLAGS          = -mod=readonly
GO_VERSION 		?= $(shell grep '^go ' go.mod | awk '{print $$2}')
BUILDER_IMAGE 	?= "ghcr.io/newrelic/coreint-automation:latest-go$(GO_VERSION)-ubuntu16.04"

all: build

build: clean test compile

clean:
	@echo "=== $(INTEGRATION) === [ clean ]: Removing binaries and coverage file..."
	@rm -rfv bin coverage.xml $(TARGET)

compile:
	@echo "=== $(INTEGRATION) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./src

cross-compile-all:
	@echo "=== $(INTEGRATION) === [ compile ]: Building cross-compiled binaries..."
	@xgo --targets=linux/amd64,linux/386,windows/amd64,windows/386,darwin/amd64,darwin/386 --dest=bin --out=$(BINARY_NAME) ./src

cross-compile-linux64:
	@echo "=== $(INTEGRATION) === [ compile ]: Building cross-compiled binaries..."
	@xgo --targets=linux/amd64 --dest=bin --out=$(BINARY_NAME) ./src

test:
	@echo "=== $(INTEGRATION) === [ test ]: Running unit tests..."
	@go test -race ./... -count=1

# rt-update-changelog runs the release-toolkit run.sh script by piping it into bash to update the CHANGELOG.md.
# It also passes down to the script all the flags added to the make target. To check all the accepted flags,
# see: https://github.com/newrelic/release-toolkit/blob/main/contrib/ohi-release-notes/run.sh
#  e.g. `make rt-update-changelog -- -v`
rt-update-changelog:
	curl "https://raw.githubusercontent.com/newrelic/release-toolkit/v1/contrib/ohi-release-notes/run.sh" | bash -s -- $(filter-out $@,$(MAKECMDGOALS))

# Include thematic Makefiles
include $(CURDIR)/build/ci.mk
include $(CURDIR)/build/release.mk

.PHONY: all build clean compile test rt-update-changelog
