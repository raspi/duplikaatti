APPNAME?=duplikaatti

# version from last tag
VERSION := $(shell git describe --abbrev=0 --always --tags)
BUILD := $(shell git rev-parse $(VERSION))
BUILDDATE := $(shell git log -1 --format=%aI $(VERSION))
BUILDFILES?=$$(find . -mindepth 1 -maxdepth 1 -type f \( -iname "*${APPNAME}-v*" -a ! -iname "*.shasums" \))
LDFLAGS := -ldflags "-s -w -X=main.VERSION=$(VERSION) -X=main.BUILD=$(BUILD) -X=main.BUILDDATE=$(BUILDDATE)"
RELEASETMPDIR := $(shell mktemp -d -t ${APPNAME}-rel-XXXXXX)
APPANDVER := ${APPNAME}-$(VERSION)
RELEASETMPAPPDIR := $(RELEASETMPDIR)/$(APPANDVER)

UPXFLAGS := -v -9
XZCOMPRESSFLAGS := --verbose --keep --compress --threads 0 --extreme -9

# https://golang.org/doc/install/source#environment
LINUX_ARCHS := amd64 arm arm64 ppc64 ppc64le
WINDOWS_ARCHS := amd64
DARWIN_ARCHS := amd64

default: build

build:
	@echo "GO BUILD..."
	@CGO_ENABLED=0 go build $(LDFLAGS) -v -o ./bin/${APPNAME} .

linux-build:
	@for arch in $(LINUX_ARCHS); do \
	  echo "GNU/Linux build... $$arch"; \
	  CGO_ENABLED=0 GOOS=linux GOARCH=$$arch go build $(LDFLAGS) -v -o ./bin/linux-$$arch/${APPNAME} . ; \
	done

darwin-build:
	@for arch in $(DARWIN_ARCHS); do \
	  echo "Darwin build... $$arch"; \
	  CGO_ENABLED=0 GOOS=darwin GOARCH=$$arch go build $(LDFLAGS) -v -o ./bin/darwin-$$arch/${APPNAME} . ; \
	done

windows-build:
	@for arch in $(WINDOWS_ARCHS); do \
	  echo "MS Windows build... $$arch"; \
	  CGO_ENABLED=0 GOOS=windows GOARCH=$$arch go build $(LDFLAGS) -v -o ./bin/windows-$$arch/${APPNAME}.exe . ; \
	done

# Compress executables
upx-pack:
	@upx $(UPXFLAGS) ./bin/linux-amd64/${APPNAME}
	@upx $(UPXFLAGS) ./bin/linux-arm/${APPNAME}
	@upx $(UPXFLAGS) ./bin/windows-amd64/${APPNAME}.exe

release: linux-build darwin-build windows-build upx-pack compress-everything shasums
	@echo "release done..."

shasums:
	@echo "Checksumming..."
	@pushd "release/${VERSION}" && shasum -a 256 $(BUILDFILES) > $(APPANDVER).shasums

# Copy common files to release directory
# Creates $(APPNAME)-$(VERSION) directory prefix where everything will be copied by compress-$OS targets
copycommon:
	@echo "Copying common files to temporary release directory '$(RELEASETMPAPPDIR)'.."
	@mkdir -p "$(RELEASETMPAPPDIR)/bin"
	@cp -v "./LICENSE" "$(RELEASETMPAPPDIR)"
	@cp -v "./README.md" "$(RELEASETMPAPPDIR)"
	@mkdir --parents "$(PWD)/release/${VERSION}"

# Compress files: GNU/Linux
compress-linux:
	@for arch in $(LINUX_ARCHS); do \
	  echo "GNU/Linux tar... $$arch"; \
	  cp -v "$(PWD)/bin/linux-$$arch/${APPNAME}" "$(RELEASETMPAPPDIR)/bin"; \
	  cd "$(RELEASETMPDIR)"; \
	  tar --numeric-owner --owner=0 --group=0 -zcvf "$(PWD)/release/${VERSION}/$(APPANDVER)-linux-$$arch.tar.gz" . ; \
	  rm "$(RELEASETMPAPPDIR)/bin/${APPNAME}"; \
	done

# Compress files: Darwin
compress-darwin:
	@for arch in $(DARWIN_ARCHS); do \
	  echo "Darwin tar... $$arch"; \
	  cp -v "$(PWD)/bin/darwin-$$arch/${APPNAME}" "$(RELEASETMPAPPDIR)/bin"; \
	  cd "$(RELEASETMPDIR)"; \
	  tar --owner=0 --group=0 -zcvf "$(PWD)/release/${VERSION}/$(APPANDVER)-darwin-$$arch.tar.gz" . ; \
	  rm "$(RELEASETMPAPPDIR)/bin/${APPNAME}"; \
	done

# Compress files: Microsoft Windows
compress-windows:
	@for arch in $(WINDOWS_ARCHS); do \
	  echo "MS Windows zip... $$arch"; \
	  cp -v "$(PWD)/bin/windows-$$arch/${APPNAME}.exe" "$(RELEASETMPAPPDIR)/bin"; \
	  cd "$(RELEASETMPAPPDIR)"; \
	  mv "LICENSE" "LICENSE.txt" && \
	  pandoc --standalone --to rtf --output LICENSE.rtf LICENSE.txt && \
	  rm "LICENSE.txt" ; \
	  cd "$(RELEASETMPDIR)" ; \
	  zip -v -9 -r -o -9 "$(PWD)/release/${VERSION}/$(APPANDVER)-windows-$$arch.zip" . ; \
	  rm "$(RELEASETMPAPPDIR)/LICENSE.rtf"; \
	  cp -v "$(PWD)/LICENSE" "$(RELEASETMPAPPDIR)" ; \
	  rm "$(RELEASETMPAPPDIR)/bin/${APPNAME}.exe"; \
	done

# Move all to temporary directory and compress with common files
compress-everything: copycommon compress-linux compress-windows
	@echo "$@ ..."
	rm -rf "$(RELEASETMPDIR)/*"

.PHONY: all clean test default