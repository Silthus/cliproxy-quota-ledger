VERSION ?= 0.1.0
GO ?= go
HOST_ARCH := $(shell $(GO) env GOARCH)
ARCH ?= $(HOST_ARCH)
CC_amd64 ?= x86_64-linux-gnu-gcc
CC_arm64 ?= aarch64-linux-gnu-gcc
LDFLAGS := -s -w -X github.com/Silthus/cliproxy-quota-ledger/internal/plugin.Version=$(VERSION)
RELEASE_ARCHES ?= arm64 amd64
RELEASE_DIR := dist/release

.PHONY: test build release clean FORCE

test:
	$(GO) vet ./...
	$(GO) test ./...

build: dist/linux/$(ARCH)/quota-ledger.so

dist/linux/%/quota-ledger.so: FORCE
	CGO_ENABLED=1 GOOS=linux GOARCH=$* CC=$(if $(filter $*,$(HOST_ARCH)),cc,$(CC_$*)) \
		$(GO) build -buildmode=c-shared -trimpath -ldflags '$(LDFLAGS)' -o $@ .
	rm -f $(@:.so=.h)

release: $(foreach arch,$(RELEASE_ARCHES),dist/linux/$(arch)/quota-ledger.so)
	rm -rf $(RELEASE_DIR) && mkdir -p $(RELEASE_DIR)
	for arch in $(RELEASE_ARCHES); do \
		cp dist/linux/$$arch/quota-ledger.so $(RELEASE_DIR)/quota-ledger_$(VERSION)_linux_$$arch.so; \
		zip -jq $(RELEASE_DIR)/quota-ledger_$(VERSION)_linux_$$arch.zip dist/linux/$$arch/quota-ledger.so; \
	done
	cd $(RELEASE_DIR) && sha256sum quota-ledger_* > checksums.txt

clean:
	rm -rf dist
