# Makefile — iterate on tools/rdr and get an OFFICIALLY STAMPED binary.
#
# `make` here must be indistinguishable from what /rdr-init produces.
# stages/00-bootstrap.md is the authority for that build; this file mirrors
# its stamp exactly:
#
#   go build -ldflags "-X main.version=$(git rev-parse --short HEAD)" \
#            -o $RDR_HOME/bin/recs .
#
# A bare `go build -o bin/recs .` leaves version="dev", which rdr-doctor 11b
# reports as "built outside the install path". That warning is correct and
# useful, so the fix is to stamp, never to teach the doctor to ignore it.
#
# THE BINARY IS `recs`, `bin/rdr` IS A SYMLINK TO IT. The projector reads
# three document tiers, so a name that is one tier's acronym misdescribes it
# — and `rdr` is already the repo, the document kind, the records dir, the
# kata project, the skills prefix and the seam vars. Every frozen record and
# evidence file that spells the command `bin/rdr` keeps running through the
# symlink; content on a terminal record is never amended.
#
# HOME IS THIS REPO, NOT $RDR_HOME. The engine is the thing being edited, so
# the output path is derived from this Makefile's own location. Honouring an
# exported RDR_HOME would write into whatever engine the shell happens to be
# pointed at — which during a two-checkout comparison is the other one.

RDR_HOME := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
SRC      := $(RDR_HOME)/tools/rdr
BIN      := $(RDR_HOME)/bin/recs
ALIAS    := $(RDR_HOME)/bin/rdr

# The stamp doctor 11b compares against `git rev-parse --short HEAD`. It is
# the bare sha with no -dirty suffix, and deliberately: `recs version` prints
# one field there and the doctor reads it with awk '{print $$2}', so a suffix
# would read as a DIFFERENT revision and report stale when the truth is
# uncommitted. Dirtiness is surfaced as a warning below instead.
STAMP := $(shell git -C $(RDR_HOME) rev-parse --short HEAD 2>/dev/null)
DIRTY := $(shell git -C $(RDR_HOME) status --porcelain 2>/dev/null | head -1)

GO ?= go

.DEFAULT_GOAL := build
.PHONY: build test check vet fmt install clean version help

## build: stamped binary at bin/recs (+ the bin/rdr alias) — what /rdr-init writes
build:
	@command -v $(GO) >/dev/null 2>&1 || { \
	  echo "stopped:no-go-toolchain — the projector needs Go (go.dev/dl)"; exit 1; }
	@test -n "$(STAMP)" || { \
	  echo "stopped:no-engine-revision — $(RDR_HOME) has no git HEAD to stamp with"; exit 1; }
	@cd $(SRC) && $(GO) build -ldflags "-X main.version=$(STAMP)" -o $(BIN) . \
	  || { echo "stopped:projector-build-failed — run the go build in $(SRC) by hand for the compiler error"; exit 1; }
	@ln -sfn recs $(ALIAS)
	@$(BIN) version
	@test -z "$(DIRTY)" || echo "note: working tree is dirty — this binary is $(STAMP) PLUS uncommitted changes"

## test: go test ./... (the suite rdr-doctor's checks stand on)
test:
	@cd $(SRC) && $(GO) test ./...

## vet: go vet ./...
vet:
	@cd $(SRC) && $(GO) vet ./...

## fmt: gofmt -w over the package
fmt:
	@cd $(SRC) && $(GO) fmt ./...

## check: build + vet + test — run before every commit that touches tools/rdr
check: build vet test
	@echo "check: ok — $$($(BIN) version)"

## install: alias for build; the binary is already at its installed path
install: build

## version: what the current binary was stamped with, vs the engine now
version:
	@test -x $(BIN) && $(BIN) version || echo "no binary at $(BIN) — run make"
	@echo "engine HEAD: $(STAMP)$(if $(DIRTY), (dirty),)"

## clean: remove the built binary and its alias
clean:
	@rm -f $(BIN) $(ALIAS)
	@echo "removed $(BIN) and $(ALIAS)"

## help: list targets
help:
	@grep -hE '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
