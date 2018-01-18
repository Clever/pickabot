include golang.mk
.DEFAULT_GOAL := test # override default goal set in library makefile

.PHONY: all test build run
SHELL := /bin/bash
PKG := github.com/Clever/pickabot
PKGS := $(shell go list ./... | grep -v /vendor | grep -v db | grep -v /mock | grep -v /slackapi)
EXECUTABLE := $(shell basename $(PKG))
$(eval $(call golang-version-check,1.9))

all: test build

test: $(PKGS)
$(PKGS): golang-test-all-strict-deps
	$(call golang-test-all-strict,$@)

build:
	$(call golang-build,$(PKG),$(EXECUTABLE))

run: build
	./bin/pickabot

install_deps: golang-dep-vendor-deps
	$(call golang-dep-vendor)

# TODO: Install predictable mockgen version into bin/
# mockgen:
# 	...

mocks:
	mkdir -p mock_slackapi && mockgen github.com/Clever/pickabot/slackapi SlackAPIService,SlackRTMService > mock_slackapi/MockSlackService.go
