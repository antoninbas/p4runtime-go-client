GO := go
BINDIR := $(CURDIR)/bin
GOLANGCI_LINT_VERSION := v1.53.3
GOLANGCI_LINT_BINDIR  := .golangci-bin
GOLANGCI_LINT_BIN     := $(GOLANGCI_LINT_BINDIR)/$(GOLANGCI_LINT_VERSION)/golangci-lint

all: bin

.bins/go-bindata:
	@mkdir -p .bins
	@GO111MODULE=off GOBIN=$(CURDIR)/.bins/ go get -u github.com/go-bindata/go-bindata/...

.bins/p4c-lite.sh:
	@mkdir -p .bins
	@curl -s https://raw.githubusercontent.com/antoninbas/p4c-lite-docker/master/p4c-lite.sh -o $@
	@chmod +x $@

l2_switch: .bins/go-bindata .bins/p4c-lite.sh
	@mkdir -p $(CURDIR)/cmd/l2_switch/l2_switch.out
	./.bins/p4c-lite.sh --pull $(CURDIR)/cmd/l2_switch/l2_switch.p4 -o $(CURDIR)/cmd/l2_switch/l2_switch.out/
	./.bins/go-bindata -o $(CURDIR)/cmd/l2_switch/l2_switch_data.go cmd/l2_switch/l2_switch.out

wcmp: .bins/go-bindata .bins/p4c-lite.sh
	@mkdir -p $(CURDIR)/cmd/wcmp/wcmp.out
	./.bins/p4c-lite.sh --pull $(CURDIR)/cmd/wcmp/wcmp.p4 -o $(CURDIR)/cmd/wcmp/wcmp.out/
	./.bins/go-bindata -o $(CURDIR)/cmd/wcmp/wcmp_data.go cmd/wcmp/wcmp.out

.PHONY: bin
bin:
	GOBIN=$(BINDIR) $(GO) install github.com/antoninbas/p4runtime-go-client/...

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

# code linting
$(GOLANGCI_LINT_BIN):
	@echo "===> Installing Golangci-lint <==="
	@rm -rf $(GOLANGCI_LINT_BINDIR)/* # delete old versions
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOLANGCI_LINT_BINDIR)/$(GOLANGCI_LINT_VERSION) $(GOLANGCI_LINT_VERSION)

.PHONY: golangci
golangci: $(GOLANGCI_LINT_BIN)
	@echo "===> Running golangci <==="
	@GOOS=linux $(GOLANGCI_LINT_BIN) run -c .golangci.yml

.PHONY: golangci-fix
golangci-fix: $(GOLANGCI_LINT_BIN)
	@echo "===> Running golangci-fix <==="
	@GOOS=linux $(GOLANGCI_LINT_BIN) run -c .golangci.yml --fix

.PHONY: clean
clean:
	rm -rf $(BINDIR)
	rm -rf .bins
	rm -rf $(GOLANGCI_LINT_BINDIR)
