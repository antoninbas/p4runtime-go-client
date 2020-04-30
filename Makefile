GO := go
BINDIR := $(CURDIR)/bin

all: bin

.PHONY: bin
bin:
	GOBIN=$(BINDIR) $(GO) install github.com/antoninbas/p4runtime-go-client/...

clean:
	rm -rf bin

.PHONY: fmt
fmt:
	$(GO) fmt github.com/antoninbas/p4runtime-go-client/...

# Run unit tests only, no integration tests
.PHONY: check-unit
check-unit:
	$(GO) test github.com/antoninbas/p4runtime-go-client/...

.PHONY: check
check:
	$(GO) test -tags=integration github.com/antoninbas/p4runtime-go-client/...
