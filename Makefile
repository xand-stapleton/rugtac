BINARY := rugtac
GO ?= go
PREFIX ?= $(HOME)/.local
INSTALL ?= install

.DEFAULT_GOAL := help

.PHONY: help build install uninstall run index test vet fmt check clean

help:
	@echo "rugtac"
	@echo
	@echo "  make build              build ./rugtac"
	@echo "  make install            install binary and local index under PREFIX"
	@echo "  make uninstall          remove the installed binary and local index"
	@echo "  make run ARGS='ring'    run with optional FLAGS and query"
	@echo "  make index              download and rebuild data/tactics.json"
	@echo "  make test               run all tests"
	@echo "  make vet                run Go's static checks"
	@echo "  make fmt                 format Go source"
	@echo "  make check              run tests and static checks"
	@echo "  make clean              remove the built binary"

build:
	$(GO) build -o $(BINARY) ./bin/rugtac

install:
	$(INSTALL) -d "$(DESTDIR)$(PREFIX)/bin" "$(DESTDIR)$(PREFIX)/share/rugtac"
	$(GO) build -o "$(DESTDIR)$(PREFIX)/bin/$(BINARY)" ./bin/rugtac
	$(INSTALL) -m 0644 data/tactics.json "$(DESTDIR)$(PREFIX)/share/rugtac/tactics.json"

uninstall:
	$(RM) "$(DESTDIR)$(PREFIX)/bin/$(BINARY)"
	$(RM) "$(DESTDIR)$(PREFIX)/share/rugtac/tactics.json"
	-rmdir "$(DESTDIR)$(PREFIX)/share/rugtac"

run:
	$(GO) run ./bin/rugtac $(FLAGS) -- $(ARGS)

index:
	$(GO) run ./bin/rugtac-index

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

check: test vet

clean:
	$(RM) $(BINARY)
