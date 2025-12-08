SHELL := /bin/bash

VERSION_FILE := internal/version/version.go

.PHONY: build
build:
	go build -o dg ./cmd/dg

.PHONY: test
test:
	go test ./...

.PHONY: release
# usage: make release VERSION=v1.2.3
release:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required, e.g., make release VERSION=v1.2.3"; exit 1; fi
	@grep -q "const Version = \"$(VERSION)\"" $(VERSION_FILE) || sed -i "s/const Version = \".*\"/const Version = \"$(VERSION)\"/" $(VERSION_FILE)
	git add $(VERSION_FILE)
	git commit -m "chore: release $(VERSION)"
	git tag $(VERSION)
	git push
	git push --tags

.PHONY: bump-patch
bump-patch:
	@v=$$(git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0); \
	IFS=. read -r major minor patch <<< $${v#v}; \
	new=v$$major.$$minor.$$((patch+1)); \
	$(MAKE) release VERSION=$$new

.PHONY: bump-minor
bump-minor:
	@v=$$(git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0); \
	IFS=. read -r major minor patch <<< $${v#v}; \
	new=v$$major.$$((minor+1)).0; \
	$(MAKE) release VERSION=$$new

.PHONY: bump-major
bump-major:
	@v=$$(git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0); \
	IFS=. read -r major minor patch <<< $${v#v}; \
	new=v$$((major+1)).0.0; \
	$(MAKE) release VERSION=$$new
