GO := go
BINDIR := $(CURDIR)/bin

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

.PHONY: bin
bin:
	GOBIN=$(BINDIR) $(GO) install github.com/antoninbas/p4runtime-go-client/...

clean:
	rm -rf bin
	rm -rf .bins
	rm -rf .golangci-bin

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
.golangci-bin:
	@echo "===> Installing Golangci-lint <==="
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $@ v1.40.0

.PHONY: golangci
golangci: .golangci-bin
	@echo "===> Running golangci <==="
	@GOOS=linux .golangci-bin/golangci-lint run -c .golangci.yml

.PHONY: golangci-fix
golangci-fix: .golangci-bin
	@echo "===> Running golangci-fix <==="
	@GOOS=linux .golangci-bin/golangci-lint run -c .golangci.yml --fix
