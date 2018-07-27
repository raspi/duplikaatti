LAST_TAG := $(shell git describe --abbrev=0 --always --tags)

BINARY := duplikaatti
BUILDDIR := build

LDFLAGS := -ldflags "-s -w -X=main.VERSION=$(LAST_TAG)"

bin:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -v -o $(BUILDDIR)/$(BINARY)

bin-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -v -o $(BUILDDIR)/$(BINARY).exe

.PHONY: all clean test
