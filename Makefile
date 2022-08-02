WORKDIR     := $(shell pwd)
TARGET      := target
TARGET_DIR   = $(WORKDIR)/$(TARGET)
NATIVEOS    := $(shell go version | awk -F '[ /]' '{print $$4}')
NATIVEARCH  := $(shell go version | awk -F '[ /]' '{print $$5}')
INTEGRATION := oracledb
BINARY_NAME  = nri-$(INTEGRATION)
GO_FILES    := ./src/
GOFLAGS      = -mod=readonly

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

# Include thematic Makefiles
include $(CURDIR)/build/ci.mk
include $(CURDIR)/build/release.mk
include $(CURDIR)/test/terraform_modules/provision.mk

.PHONY: all build clean compile test
