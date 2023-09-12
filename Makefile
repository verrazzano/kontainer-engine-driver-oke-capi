# Copyright (c) 2023, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

NAME:=kontainer-engine-driver-okecapi

DRIVER_NAME:=kontainer-engine-driver-okecapi

# local build, use user and timestamp it
BINARY_NAME ?= ${NAME}
VERSION:=$(shell  date +%Y%m%d%H%M%S)
DIST_DIR:=dist
GO ?= go

#
# Go build related tasks
#
.PHONY: go-install
go-install:
	GO111MODULE=on $(GO) install .

.PHONY: go-run
go-run: go-install
	GO111MODULE=on $(GO) run .

.PHONY: go-fmt
go-fmt:
	gofmt -s -e -d $(shell find . -name "*.go" | grep -v /vendor/)

.PHONY: go-vet
go-vet:
	echo $(GO) vet $(shell $(GO) list ./... | grep -v /vendor/)

.PHONY: build
build:
	rm -rf ${DIST_DIR}
	mkdir -p ${DIST_DIR}
	GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o ${DIST_DIR}/${BINARY_NAME}-linux .

.PHONY: sha256sum
sha256sum: build
	shasum -a 256 dist/kontainer-engine-driver-okecapi-linux

.PHONY: shasum
shasum: build
	sha256sum dist/kontainer-engine-driver-okecapi-linux

.PHONY: cr
cr:
	./scripts/write_cr.sh

#
# Tests-related tasks
#
.PHONY: unit-test
unit-test: go-install
	go test -v ./pkg/...

.PHONY: lint
lint:
ifeq (, $(shell command -v golangci-lint))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.49.0
endif
	golangci-lint run --skip-dirs=verrazzano

.PHONY: copyright-check
copyright-check: checkout-copyright-repo
	go run ./verrazzano/tools/copyright --enforce-current .

.PHONY: checkout-copyright-repo
checkout-copyright-repo:
	rm -rf verrazzano
	git clone -n --depth=1 --filter=tree:0 https://github.com/verrazzano/verrazzano
	cd verrazzano && git sparse-checkout set --no-cone tools/copyright && git checkout

.PHONY: ci
ci: | copyright-check lint unit-test build
