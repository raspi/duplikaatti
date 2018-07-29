LAST_TAG := $(shell git describe --abbrev=0 --always --tags)
BUILD := $(shell git rev-parse $(LAST_TAG))

BINARY := duplikaatti
UNIXBINARY := $(BINARY)-x64
WINBINARY := $(UNIXBINARY).exe
BUILDDIR := build

LINUXRELEASE := $(BINARY)-$(LAST_TAG)-linux-x64.tar.gz
WINRELEASE := $(BINARY)-$(LAST_TAG)-windows-x64.zip

LDFLAGS := -ldflags "-s -w -X=main.VERSION=$(LAST_TAG) -X=main.BUILD=$(BUILD)"

bin:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -v -o $(BUILDDIR)/$(UNIXBINARY)
	upx -v -9 $(BUILDDIR)/$(UNIXBINARY)

bin-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -v -o $(BUILDDIR)/$(WINBINARY)
	upx -v -9 $(BUILDDIR)/$(WINBINARY)

release:
	tar cvzf $(BUILDDIR)/$(LINUXRELEASE) $(BUILDDIR)/$(UNIXBINARY)

release-windows:
	zip -v -9 $(BUILDDIR)/$(WINRELEASE) $(BUILDDIR)/$(WINBINARY)

.PHONY: all clean test
