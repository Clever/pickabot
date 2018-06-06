include golang.mk
.DEFAULT_GOAL := test # override default goal set in library makefile

.PHONY: all test build run
SHELL := /bin/bash
PKG := github.com/Clever/pickabot
PKGS := $(shell go list ./... | grep -v /vendor | grep -v db | grep -v /mock | grep -v /slackapi)
EXECUTABLE := $(shell basename $(PKG))
$(eval $(call golang-version-check,1.10))

all: test build

test: generate $(PKGS)
$(PKGS): golang-test-all-strict-deps
	$(call golang-test-all-strict,$@)

build: 
	$(call golang-build,$(PKG),$(EXECUTABLE))

run: build
	./bin/pickabot

install_deps: golang-dep-vendor-deps
	$(call golang-dep-vendor)
	go build -o bin/mockgen ./vendor/github.com/golang/mock/mockgen

generate:
	go generate $(PKG)/mocks

